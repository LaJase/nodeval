package validator_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"nodeval/internal/validator"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// stubLoader is a test double that returns a pre-compiled schema.
type stubLoader struct {
	schema *jsonschema.Schema
}

func (s *stubLoader) Load(_ string) (*jsonschema.Schema, error) {
	return s.schema, nil
}

func makeSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	dir := t.TempDir()
	path := filepath.Join(dir, "s.json")
	_ = os.WriteFile(path, []byte(`{"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}`), 0o644)
	sch, err := compiler.Compile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sch
}

func makeNestedSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	dir := t.TempDir()
	path := filepath.Join(dir, "s.json")
	src := `{
		"type": "object",
		"properties": {
			"users": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"age": {"type": "integer", "minimum": 18}
					},
					"required": ["age"]
				}
			}
		}
	}`
	_ = os.WriteFile(path, []byte(src), 0o644)
	sch, err := compiler.Compile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sch
}

func writeJSON(t *testing.T, dir, name string, v any) string {
	t.Helper()
	b, _ := json.Marshal(v)
	p := filepath.Join(dir, name)
	_ = os.WriteFile(p, b, 0o644)
	return p
}

func TestRun_AllValidFiles(t *testing.T) {
	dir := t.TempDir()
	sch := makeSchema(t)
	loader := &stubLoader{schema: sch}

	const n = 50
	files := make([]string, n)
	for i := range files {
		files[i] = writeJSON(t, dir, fmt.Sprintf("file_%d_T.json", i), map[string]int{"id": i})
	}

	results := validator.Run(map[string][]string{"T": files}, loader, validator.Options{Workers: 4})

	if len(results) != 1 {
		t.Fatalf("expected 1 type result, got %d", len(results))
	}
	if results[0].Success != n {
		t.Errorf("expected %d successes, got %d", n, results[0].Success)
	}
	if results[0].Errors != 0 {
		t.Errorf("expected 0 errors, got %d", results[0].Errors)
	}
}

func TestRun_OnProgressTotalMatchesFileCount(t *testing.T) {
	dir := t.TempDir()
	sch := makeSchema(t)
	loader := &stubLoader{schema: sch}

	const n = 200
	files := make([]string, n)
	for i := range files {
		files[i] = writeJSON(t, dir, fmt.Sprintf("file_%d_T.json", i), map[string]int{"id": i})
	}

	var mu sync.Mutex
	total := 0
	calls := 0

	validator.Run(map[string][]string{"T": files}, loader, validator.Options{
		Workers: 4,
		OnProgress: func(_ string, count int) {
			mu.Lock()
			total += count
			calls++
			mu.Unlock()
		},
	})

	if total != n {
		t.Errorf("OnProgress total = %d, want %d", total, n)
	}
	if calls >= n {
		t.Errorf("OnProgress called %d times for %d files: batching not working", calls, n)
	}
}

func TestRun_ErrorPath_UsesDotBracketNotation(t *testing.T) {
	dir := t.TempDir()
	sch := makeNestedSchema(t)
	loader := &stubLoader{schema: sch}

	// users[0].age violates minimum:18
	file := writeJSON(t, dir, "bad_T.json", map[string]any{
		"users": []map[string]any{{"age": 12}},
	})

	results := validator.Run(map[string][]string{"T": {file}}, loader, validator.Options{Workers: 1})

	if len(results) != 1 || len(results[0].Details) == 0 {
		t.Fatal("expected one error detail")
	}
	got := results[0].Details[0].Path
	want := "users[0].age"
	if got != want {
		t.Errorf("Path = %q, want %q", got, want)
	}
}

func TestRun_ErrorPath_MissingRequiredField(t *testing.T) {
	dir := t.TempDir()
	sch := makeNestedSchema(t)
	loader := &stubLoader{schema: sch}

	// users[0] missing required "age"
	file := writeJSON(t, dir, "bad_T.json", map[string]any{
		"users": []map[string]any{{"name": "Alice"}},
	})

	results := validator.Run(map[string][]string{"T": {file}}, loader, validator.Options{Workers: 1})

	if len(results) != 1 || len(results[0].Details) == 0 {
		t.Fatal("expected one error detail")
	}
	got := results[0].Details[0].Path
	want := "users[0]"
	if got != want {
		t.Errorf("Path = %q, want %q", got, want)
	}
}

func TestRun_MixedValidInvalidFiles(t *testing.T) {
	dir := t.TempDir()
	sch := makeSchema(t)
	loader := &stubLoader{schema: sch}

	var files []string
	for i := range 30 {
		files = append(files, writeJSON(t, dir, fmt.Sprintf("ok_%d_T.json", i), map[string]int{"id": i}))
	}
	for i := range 20 {
		// missing required "id" field
		files = append(files, writeJSON(t, dir, fmt.Sprintf("bad_%d_T.json", i), map[string]string{"name": "x"}))
	}

	results := validator.Run(map[string][]string{"T": files}, loader, validator.Options{Workers: 4})

	if results[0].Success != 30 {
		t.Errorf("expected 30 successes, got %d", results[0].Success)
	}
	if results[0].Errors != 20 {
		t.Errorf("expected 20 errors, got %d", results[0].Errors)
	}
}

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
