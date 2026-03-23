# Error Descriptions: Verbose vs Normal Mode

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Différencier l'affichage en mode normal (compte d'erreurs par fichier) et verbose (toutes les erreurs groupées par fichier) sans dégrader les perfs en mode normal.

**Architecture:** Ajout d'un champ `Count` à `FileError`, deux chemins d'extraction dans `validateFile` selon `verbose bool`, et adaptation du rendu terminal pour grouper les erreurs par fichier en mode verbose.

**Tech Stack:** Go, `github.com/santhosh-tekuri/jsonschema/v5`, `github.com/fatih/color`

---

### Task 1: Add `Count` to `FileError` and extract `formatMessage`

**Files:**
- Modify: `internal/validator/validator.go:18-22` (struct FileError)
- Modify: `internal/validator/validator.go:194-207` (extractError + new helper)

**Step 1: Add `Count int` to `FileError`**

In `validator.go`, replace the `FileError` struct:

```go
type FileError struct {
	File    string `json:"file"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message,omitempty"`
	Count   int    `json:"count,omitempty"` // >1 when multiple errors exist but details were not extracted (normal mode)
}
```

**Step 2: Extract `formatMessage` from `extractError`**

Add the helper just above `extractError`, and simplify `extractError` to use it:

```go
// formatMessage applies human-friendly transformations to a raw schema error message.
func formatMessage(msg string) string {
	if props, ok := strings.CutPrefix(msg, "missing properties: "); ok {
		return fmt.Sprintf("%s are required", strings.ReplaceAll(props, "'", ""))
	}
	return msg
}

func extractError(ve *jsonschema.ValidationError) (path, msg string) {
	curr := ve
	for len(curr.Causes) > 0 {
		curr = curr.Causes[0]
	}
	return jsonPtrToDot(curr.InstanceLocation), formatMessage(curr.Message)
}
```

**Step 3: Run tests to make sure nothing is broken**

```bash
go test ./...
```

Expected: all tests pass.

**Step 4: Commit**

```bash
git add internal/validator/validator.go
git commit -m "refactor: add Count to FileError, extract formatMessage helper"
```

---

### Task 2: Add `countLeafErrors`

**Files:**
- Modify: `internal/validator/validator.go`
- Modify: `internal/validator/validator_test.go`

**Step 1: Write the failing test**

Add `makeMultiErrorSchema` helper and the test to `validator_test.go`:

```go
func makeMultiErrorSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	dir := t.TempDir()
	path := filepath.Join(dir, "s.json")
	src := `{
		"type": "object",
		"properties": {
			"id":   {"type": "integer"},
			"name": {"type": "string"}
		}
	}`
	_ = os.WriteFile(path, []byte(src), 0o644)
	sch, err := compiler.Compile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sch
}

func TestRun_NormalMode_MultipleErrors_ShowsCount(t *testing.T) {
	dir := t.TempDir()
	sch := makeMultiErrorSchema(t)
	loader := &stubLoader{schema: sch}

	// id and name both have wrong types → 2 leaf errors
	file := writeJSON(t, dir, "bad_T.json", map[string]any{
		"id":   "not-an-int",
		"name": 42,
	})

	results := validator.Run(map[string][]string{"T": {file}}, loader, validator.Options{Workers: 1})

	if len(results) != 1 || len(results[0].Details) == 0 {
		t.Fatal("expected one error detail")
	}
	d := results[0].Details[0]
	if d.Count <= 1 {
		t.Errorf("expected Count > 1 in normal mode, got Count=%d", d.Count)
	}
	if d.Path != "" || d.Message != "" {
		t.Errorf("expected no path/message in normal mode multi-error, got Path=%q Message=%q", d.Path, d.Message)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/validator/... -run TestRun_NormalMode_MultipleErrors_ShowsCount -v
```

Expected: FAIL (Count is 0 or Path/Message are set).

**Step 3: Implement `countLeafErrors` in `validator.go`**

Add just above `extractError`:

```go
// countLeafErrors returns the number of leaf causes in a ValidationError tree.
// It performs no string allocations, making it safe to call in normal (non-verbose) mode.
func countLeafErrors(ve *jsonschema.ValidationError) int {
	if len(ve.Causes) == 0 {
		return 1
	}
	n := 0
	for _, c := range ve.Causes {
		n += countLeafErrors(c)
	}
	return n
}
```

This function will be used in Task 4, but writing it now lets the test pass once validateFile is updated.

**Step 4: Run tests**

```bash
go test ./internal/validator/... -v
```

Expected: `TestRun_NormalMode_MultipleErrors_ShowsCount` still fails (validateFile not yet updated), all others pass.

**Step 5: Commit**

```bash
git add internal/validator/validator.go internal/validator/validator_test.go
git commit -m "feat: add countLeafErrors + test for normal mode multi-error count"
```

---

### Task 3: Add `extractAllErrors`

**Files:**
- Modify: `internal/validator/validator.go`
- Modify: `internal/validator/validator_test.go`

**Step 1: Write the failing test**

Add to `validator_test.go`:

```go
func TestRun_VerboseMode_MultipleErrors_ShowsAll(t *testing.T) {
	dir := t.TempDir()
	sch := makeMultiErrorSchema(t)
	loader := &stubLoader{schema: sch}

	file := writeJSON(t, dir, "bad_T.json", map[string]any{
		"id":   "not-an-int",
		"name": 42,
	})

	results := validator.Run(map[string][]string{"T": {file}}, loader, validator.Options{
		Workers: 1,
		Verbose: true,
	})

	if len(results) != 1 {
		t.Fatal("expected one type result")
	}
	if got := len(results[0].Details); got != 2 {
		t.Fatalf("expected 2 error details in verbose mode, got %d", got)
	}
	for _, d := range results[0].Details {
		if d.File == "" {
			t.Error("expected File to be set on every detail")
		}
		if d.Path == "" || d.Message == "" {
			t.Errorf("expected Path and Message to be set in verbose mode, got Path=%q Message=%q", d.Path, d.Message)
		}
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/validator/... -run TestRun_VerboseMode_MultipleErrors_ShowsAll -v
```

Expected: FAIL (only 1 detail, not 2).

**Step 3: Implement `extractAllErrors` in `validator.go`**

Add after `extractError`:

```go
// extractAllErrors traverses all leaf causes and returns one FileError per leaf.
// File is left empty; the caller sets it.
func extractAllErrors(ve *jsonschema.ValidationError) []FileError {
	if len(ve.Causes) == 0 {
		path := jsonPtrToDot(ve.InstanceLocation)
		msg := formatMessage(ve.Message)
		return []FileError{{Path: path, Message: msg}}
	}
	var result []FileError
	for _, c := range ve.Causes {
		result = append(result, extractAllErrors(c)...)
	}
	return result
}
```

Tests will pass once `validateFile` is updated in Task 4.

**Step 4: Run tests**

```bash
go test ./internal/validator/... -v
```

Expected: all existing tests pass, new tests still fail (validateFile not yet updated).

**Step 5: Commit**

```bash
git add internal/validator/validator.go internal/validator/validator_test.go
git commit -m "feat: add extractAllErrors for verbose multi-error extraction"
```

---

### Task 4: Update `validateFile` and the worker loop

**Files:**
- Modify: `internal/validator/validator.go`

**Step 1: Update `localBatch.add` to accept `[]FileError`**

Replace the `add` method (currently at line ~50):

```go
func (b *localBatch) add(ok bool, fes []FileError) {
	if ok {
		b.success++
	} else {
		b.errors++
		b.details = append(b.details, fes...)
	}
}
```

**Step 2: Update `validateFile` signature and body**

Replace the full `validateFile` function:

```go
func validateFile(sch *jsonschema.Schema, fPath string, verbose bool) ([]FileError, bool) {
	baseName := filepath.Base(fPath)

	data, err := os.ReadFile(fPath)
	if err != nil {
		return []FileError{{File: baseName, Message: fmt.Sprintf("read error: %v", err)}}, false
	}

	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return []FileError{{File: baseName, Message: fmt.Sprintf("invalid JSON: %v", err)}}, false
	}

	errVal := sch.Validate(v)
	if errVal == nil {
		return nil, true
	}

	ve, ok := errVal.(*jsonschema.ValidationError)
	if !ok {
		return []FileError{{File: baseName, Message: fmt.Sprintf("%v", errVal)}}, false
	}

	if verbose {
		fes := extractAllErrors(ve)
		for i := range fes {
			fes[i].File = baseName
		}
		return fes, false
	}

	// Normal mode: avoid allocating path/message strings when there are multiple errors.
	count := countLeafErrors(ve)
	if count == 1 {
		errPath, msg := extractError(ve)
		return []FileError{{File: baseName, Path: errPath, Message: msg}}, false
	}
	return []FileError{{File: baseName, Count: count}}, false
}
```

**Step 3: Update the worker loop to use the new signatures**

In the `Run` function, find the two call sites that use `validateFile` and `localBatch.add`:

The "missing schema" error (around line 128) — change to pass a slice:
```go
batches[t.typeNode].add(false, []FileError{{
    File:    t.typeNode,
    Message: fmt.Sprintf("missing schema: %s", t.typeNode),
}})
```

The `validateFile` call (around line 142):
```go
fes, ok := validateFile(sch, t.path, opts.Verbose)
batches[t.typeNode].add(ok, fes)
```

**Step 4: Run all tests**

```bash
go test ./internal/validator/... -v
```

Expected: all tests pass including the two new ones from Tasks 2 and 3.

**Step 5: Commit**

```bash
git add internal/validator/validator.go
git commit -m "feat: validateFile respects verbose flag — count-only in normal, all errors in verbose"
```

---

### Task 5: Update terminal renderer

**Files:**
- Modify: `internal/reporter/terminal.go`
- Modify: `internal/reporter/terminal_test.go`

**Step 1: Write tests for the new display logic**

Add to `terminal_test.go`:

```go
import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"nodeval/internal/reporter"
	"nodeval/internal/validator"
)

func captureStdout(f func()) string {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestTerminal_NormalMode_SingleError_ShowsPathAndMessage(t *testing.T) {
	report := reporter.Report{
		Duration: time.Second,
		Results: []validator.TypeResult{
			{Type: "T", Success: 0, Errors: 1, Details: []validator.FileError{
				{File: "foo_T.json", Path: "id", Message: "expected integer"},
			}},
		},
	}
	out := captureStdout(func() {
		tr := &reporter.Terminal{Verbose: false}
		_ = tr.Render(report)
	})
	if !strings.Contains(out, "foo_T.json") {
		t.Error("expected file name in output")
	}
	if !strings.Contains(out, "id") {
		t.Error("expected path in output")
	}
	if !strings.Contains(out, "expected integer") {
		t.Error("expected message in output")
	}
}

func TestTerminal_NormalMode_MultipleErrors_ShowsCount(t *testing.T) {
	report := reporter.Report{
		Duration: time.Second,
		Results: []validator.TypeResult{
			{Type: "T", Success: 0, Errors: 1, Details: []validator.FileError{
				{File: "bar_T.json", Count: 3},
			}},
		},
	}
	out := captureStdout(func() {
		tr := &reporter.Terminal{Verbose: false}
		_ = tr.Render(report)
	})
	if !strings.Contains(out, "bar_T.json") {
		t.Error("expected file name in output")
	}
	if !strings.Contains(out, "3 errors") {
		t.Errorf("expected '3 errors' in output, got:\n%s", out)
	}
}

func TestTerminal_VerboseMode_GroupsErrorsByFile(t *testing.T) {
	report := reporter.Report{
		Duration: time.Second,
		Results: []validator.TypeResult{
			{Type: "T", Success: 0, Errors: 1, Details: []validator.FileError{
				{File: "baz_T.json", Path: "id", Message: "expected integer"},
				{File: "baz_T.json", Path: "name", Message: "expected string"},
			}},
		},
	}
	out := captureStdout(func() {
		tr := &reporter.Terminal{Verbose: true}
		_ = tr.Render(report)
	})
	// File name should appear once, errors indented below
	fileCount := strings.Count(out, "baz_T.json")
	if fileCount != 1 {
		t.Errorf("expected file name once in verbose output, got %d occurrences", fileCount)
	}
	if !strings.Contains(out, "id") || !strings.Contains(out, "name") {
		t.Error("expected both error paths in verbose output")
	}
}
```

**Step 2: Run tests to verify they fail**

```bash
go test ./internal/reporter/... -run "TestTerminal_" -v
```

Expected: failures on the new tests.

**Step 3: Update `Render` in `terminal.go`**

Replace the error-display loop (lines 45-63) with:

```go
for _, res := range r.Results {
    if t.Verbose {
        // Group FileErrors by file (preserve order of first appearance).
        type group struct {
            file   string
            errors []validator.FileError
        }
        seen := make(map[string]int)
        var groups []group
        for _, d := range res.Details {
            if i, ok := seen[d.File]; ok {
                groups[i].errors = append(groups[i].errors, d)
            } else {
                seen[d.File] = len(groups)
                groups = append(groups, group{file: d.File, errors: []validator.FileError{d}})
            }
        }
        for _, g := range groups {
            fmt.Printf("%s %s :\n", color.RedString("❌"), color.YellowString(g.file))
            for _, e := range g.errors {
                fmt.Printf("   %s : %s\n", e.Path, e.Message)
            }
            fmt.Println()
        }
    } else {
        for _, d := range res.Details {
            if d.Count > 1 {
                fmt.Printf("%s %s : %d errors\n",
                    color.RedString("❌"),
                    color.YellowString(d.File),
                    d.Count,
                )
            } else {
                fmt.Printf("%s %s : %s : %s\n",
                    color.RedString("❌"),
                    color.YellowString(d.File),
                    d.Path,
                    d.Message,
                )
            }
        }
    }
}
```

**Step 4: Run all tests**

```bash
go test ./... -v
```

Expected: all tests pass.

**Step 5: Commit**

```bash
git add internal/reporter/terminal.go internal/reporter/terminal_test.go
git commit -m "feat: terminal renderer groups errors by file in verbose, shows count in normal mode"
```

---

### Task 6: Final verification

**Step 1: Run full test suite**

```bash
go test ./... -race
```

Expected: all tests pass with no race conditions.

**Step 2: Build and smoke test**

```bash
go build -o nodeval . && echo "build ok"
```

**Step 3: Commit if any fixes needed, then done**
