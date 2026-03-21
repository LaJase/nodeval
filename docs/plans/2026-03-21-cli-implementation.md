# jsnsch CLI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Transformer le `main.go` existant en un CLI professionnel avec Cobra/Viper, sous-commandes, multi-format de sortie, et détection automatique des types.

**Architecture:** Structure de packages Go standard — `cmd/` pour les commandes Cobra, `internal/` pour toute la logique métier (config, scanner, schema, validator, reporter). Le code existant de `main.go` est découpé et réparti dans les packages appropriés.

**Tech Stack:** Go 1.26, Cobra v1, Viper v2, fatih/color, vbauerster/mpb, santhosh-tekuri/jsonschema/v5, encoding/xml (stdlib)

---

### Task 1: Ajouter les dépendances et créer la structure de répertoires

**Files:**
- Modify: `go.mod`
- Create: `cmd/`, `internal/config/`, `internal/scanner/`, `internal/schema/`, `internal/validator/`, `internal/reporter/` (répertoires)

**Step 1: Ajouter cobra et viper**

```bash
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest
```

**Step 2: Créer la structure de répertoires**

```bash
mkdir -p cmd internal/config internal/scanner internal/schema internal/validator internal/reporter
```

**Step 3: Vérifier go.mod**

```bash
go mod tidy
cat go.mod
```
Expected: cobra et viper présents dans les requires.

**Step 4: Commit**

```bash
git init
git add go.mod go.sum
git commit -m "chore: add cobra and viper dependencies"
```

---

### Task 2: internal/config — Struct Config + chargement Viper

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Écrire le test**

```go
// internal/config/config_test.go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"jsnsch/internal/config"
)

func TestDefaults(t *testing.T) {
	cfg := config.Default()
	if cfg.Output != "terminal" {
		t.Errorf("expected output=terminal, got %s", cfg.Output)
	}
	if cfg.Workers != 0 {
		t.Errorf("expected workers=0, got %d", cfg.Workers)
	}
	if cfg.Verbose {
		t.Error("expected verbose=false")
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	content := []byte("output: json\nworkers: 4\nverbose: true\n")
	_ = os.WriteFile(filepath.Join(dir, ".jsnsch.yaml"), content, 0644)

	cfg, err := config.LoadFrom(filepath.Join(dir, ".jsnsch.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Output != "json" {
		t.Errorf("expected output=json, got %s", cfg.Output)
	}
	if cfg.Workers != 4 {
		t.Errorf("expected workers=4, got %d", cfg.Workers)
	}
}
```

**Step 2: Vérifier que le test échoue**

```bash
go test ./internal/config/...
```
Expected: FAIL — package not found.

**Step 3: Implémenter**

```go
// internal/config/config.go
package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Directory  string
	Schemas    string   `mapstructure:"schemas"`
	Types      []string `mapstructure:"types"`
	All        bool     `mapstructure:"all"`
	Output     string   `mapstructure:"output"`
	Verbose    bool     `mapstructure:"verbose"`
	Workers    int      `mapstructure:"workers"`
	NoProgress bool     `mapstructure:"no_progress"`
}

func Default() Config {
	return Config{
		Schemas:    ".",
		Output:     "terminal",
		Workers:    0,
		Verbose:    false,
		NoProgress: false,
	}
}

func LoadFrom(path string) (Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return Default(), err
	}
	cfg := Default()
	if err := v.Unmarshal(&cfg); err != nil {
		return Default(), err
	}
	return cfg, nil
}
```

**Step 4: Vérifier que les tests passent**

```bash
go test ./internal/config/... -v
```
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add config package with viper loading"
```

---

### Task 3: internal/schema — Interface Loader + LocalLoader

**Files:**
- Create: `internal/schema/loader.go`
- Create: `internal/schema/loader_test.go`

**Step 1: Écrire le test**

```go
// internal/schema/loader_test.go
package schema_test

import (
	"os"
	"path/filepath"
	"testing"

	"jsnsch/internal/schema"
)

func TestLocalLoaderMissing(t *testing.T) {
	loader := schema.NewLocalLoader(t.TempDir())
	_, err := loader.Load("X")
	if err == nil {
		t.Error("expected error for missing schema")
	}
}

func TestLocalLoaderValid(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`{"type": "object"}`)
	_ = os.WriteFile(filepath.Join(dir, "json-schema-Node_M.json"), content, 0644)

	loader := schema.NewLocalLoader(dir)
	sch, err := loader.Load("M")
	if err != nil {
		t.Fatal(err)
	}
	if sch == nil {
		t.Error("expected non-nil schema")
	}
}
```

**Step 2: Vérifier que le test échoue**

```bash
go test ./internal/schema/...
```
Expected: FAIL.

**Step 3: Implémenter**

```go
// internal/schema/loader.go
package schema

import (
	"fmt"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Loader interface {
	Load(typeNode string) (*jsonschema.Schema, error)
}

type LocalLoader struct {
	dir string
}

func NewLocalLoader(dir string) *LocalLoader {
	return &LocalLoader{dir: dir}
}

func (l *LocalLoader) Load(typeNode string) (*jsonschema.Schema, error) {
	path := filepath.Join(l.dir, fmt.Sprintf("json-schema-Node_%s.json", typeNode))
	compiler := jsonschema.NewCompiler()
	sch, err := compiler.Compile(path)
	if err != nil {
		return nil, fmt.Errorf("schema %s: %w", typeNode, err)
	}
	return sch, nil
}
```

**Step 4: Vérifier que les tests passent**

```bash
go test ./internal/schema/... -v
```
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/schema/
git commit -m "feat: add schema loader interface and local implementation"
```

---

### Task 4: internal/schema — Détection automatique des types

**Files:**
- Create: `internal/schema/detect.go`
- Create: `internal/schema/detect_test.go`

**Step 1: Écrire le test**

```go
// internal/schema/detect_test.go
package schema_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"jsnsch/internal/schema"
)

func TestDetectTypes(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"json-schema-Node_M.json",
		"json-schema-Node_R.json",
		"unrelated.json",
	} {
		_ = os.WriteFile(filepath.Join(dir, name), []byte(`{}`), 0644)
	}

	types, err := schema.DetectTypes(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 2 {
		t.Errorf("expected 2 types, got %d: %v", len(types), types)
	}
	if !slices.Contains(types, "M") || !slices.Contains(types, "R") {
		t.Errorf("expected M and R, got %v", types)
	}
}
```

**Step 2: Vérifier que le test échoue**

```bash
go test ./internal/schema/... -run TestDetectTypes
```
Expected: FAIL.

**Step 3: Implémenter**

```go
// internal/schema/detect.go
package schema

import (
	"os"
	"sort"
	"strings"
)

const schemaPrefix = "json-schema-Node_"

// DetectTypes scans dir and returns all type names found from schema filenames.
func DetectTypes(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var types []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, schemaPrefix) && strings.HasSuffix(name, ".json") {
			t := strings.TrimPrefix(name, schemaPrefix)
			t = strings.TrimSuffix(t, ".json")
			if t != "" {
				types = append(types, t)
			}
		}
	}
	sort.Strings(types)
	return types, nil
}
```

**Step 4: Vérifier que les tests passent**

```bash
go test ./internal/schema/... -v
```
Expected: PASS (les deux tests).

**Step 5: Commit**

```bash
git add internal/schema/detect.go internal/schema/detect_test.go
git commit -m "feat: add automatic type detection from schema directory"
```

---

### Task 5: internal/scanner — Scan de fichiers par type

**Files:**
- Create: `internal/scanner/scanner.go`
- Create: `internal/scanner/scanner_test.go`

**Step 1: Écrire le test**

```go
// internal/scanner/scanner_test.go
package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"jsnsch/internal/scanner"
)

func TestScanFiles(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		"node_M.json",
		"other_M.json",
		"node_R.json",
		"node_I.json",
		"unrelated.txt",
	}
	for _, f := range files {
		_ = os.WriteFile(filepath.Join(dir, f), []byte(`{}`), 0644)
	}

	result, err := scanner.ScanFiles(dir, []string{"M", "R"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result["M"]) != 2 {
		t.Errorf("expected 2 M files, got %d", len(result["M"]))
	}
	if len(result["R"]) != 1 {
		t.Errorf("expected 1 R file, got %d", len(result["R"]))
	}
	if _, ok := result["I"]; ok {
		t.Error("expected no I files (not in requested types)")
	}
}
```

**Step 2: Vérifier que le test échoue**

```bash
go test ./internal/scanner/...
```
Expected: FAIL.

**Step 3: Implémenter**

```go
// internal/scanner/scanner.go
package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

// ScanFiles walks dir recursively and returns files matching *_<type>.json per type.
func ScanFiles(dir string, types []string) (map[string][]string, error) {
	filesByType := make(map[string][]string)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		for _, t := range types {
			if strings.HasSuffix(d.Name(), "_"+t+".json") {
				filesByType[t] = append(filesByType[t], path)
			}
		}
		return nil
	})
	return filesByType, err
}
```

**Step 4: Vérifier que les tests passent**

```bash
go test ./internal/scanner/... -v
```
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/scanner/
git commit -m "feat: add file scanner package"
```

---

### Task 6: internal/validator — Worker pool et validation

**Files:**
- Create: `internal/validator/validator.go`

> Note: Ce package extrait et refactorise la logique worker + validation de `main.go`. Pas de tests unitaires pour la validation elle-même (dépend de fichiers et schémas réels) — couvert par les tests d'intégration plus tard.

**Step 1: Implémenter**

```go
// internal/validator/validator.go
package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"jsnsch/internal/schema"
)

type FileError struct {
	File    string `json:"file"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

type TypeResult struct {
	Type    string      `json:"type"`
	Success int         `json:"success"`
	Errors  int         `json:"errors"`
	Details []FileError `json:"details"`
}

type result struct {
	mu      sync.Mutex
	TypeResult
}

func (r *result) record(ok bool, fe FileError) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ok {
		r.Success++
	} else {
		r.Errors++
		r.Details = append(r.Details, fe)
	}
}

type task struct {
	path     string
	typeNode string
}

type Options struct {
	Workers    int
	Verbose    bool
	OnProgress func(typeNode string) // called after each file
}

// Run validates all files in filesByType using the provided schema loader.
func Run(filesByType map[string][]string, loader schema.Loader, opts Options) []TypeResult {
	numWorkers := opts.Workers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	totalTasks := 0
	for _, files := range filesByType {
		totalTasks += len(files)
	}

	taskChan := make(chan task, totalTasks)
	resultsMap := make(map[string]*result)

	for typeNode, files := range filesByType {
		resultsMap[typeNode] = &result{TypeResult: TypeResult{Type: typeNode}}
		for _, p := range files {
			taskChan <- task{path: p, typeNode: typeNode}
		}
	}
	close(taskChan)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			schemaCache := make(map[string]*jsonschema.Schema)
			schemaFailed := make(map[string]bool)

			for t := range taskChan {
				if schemaFailed[t.typeNode] {
					if opts.OnProgress != nil {
						opts.OnProgress(t.typeNode)
					}
					continue
				}

				sch, exists := schemaCache[t.typeNode]
				if !exists {
					var err error
					sch, err = loader.Load(t.typeNode)
					if err != nil {
						schemaFailed[t.typeNode] = true
						resultsMap[t.typeNode].record(false, FileError{
							File:    t.typeNode,
							Path:    "",
							Message: color.RedString("schéma manquant : %s", t.typeNode),
						})
						if opts.OnProgress != nil {
							opts.OnProgress(t.typeNode)
						}
						continue
					}
					schemaCache[t.typeNode] = sch
				}

				fe, ok := validateFile(sch, t.path, opts.Verbose)
				resultsMap[t.typeNode].record(ok, fe)
				if opts.OnProgress != nil {
					opts.OnProgress(t.typeNode)
				}
			}
		}()
	}
	wg.Wait()

	out := make([]TypeResult, 0, len(resultsMap))
	for _, r := range resultsMap {
		out = append(out, r.TypeResult)
	}
	return out
}

func validateFile(sch *jsonschema.Schema, fPath string, verbose bool) (FileError, bool) {
	baseName := filepath.Base(fPath)

	f, err := os.Open(fPath)
	if err != nil {
		return FileError{File: baseName, Message: "erreur lecture"}, false
	}
	defer f.Close()

	var v any
	if err := json.NewDecoder(f).Decode(&v); err != nil {
		return FileError{File: baseName, Message: "JSON malformé"}, false
	}

	errVal := sch.Validate(v)
	if errVal == nil {
		return FileError{}, true
	}

	ve, ok := errVal.(*jsonschema.ValidationError)
	if !ok {
		return FileError{File: baseName, Message: fmt.Sprintf("%v", errVal)}, false
	}

	errPath, msg := extractError(ve, verbose)
	return FileError{File: baseName, Path: errPath, Message: msg}, false
}

func extractError(ve *jsonschema.ValidationError, verbose bool) (path, msg string) {
	var contexts []string
	curr := ve
	for len(curr.Causes) > 0 {
		m := curr.Message
		if m != "" && !strings.Contains(m, "file://") {
			clean := strings.TrimPrefix(m, "doesn't validate with ")
			clean = strings.ReplaceAll(clean, "'", "")
			clean = strings.TrimPrefix(clean, "/definitions/")
			contexts = append(contexts, clean)
		}
		curr = curr.Causes[0]
	}

	finalMsg := curr.Message
	if strings.HasPrefix(finalMsg, "missing properties: ") {
		props := strings.TrimPrefix(finalMsg, "missing properties: ")
		finalMsg = fmt.Sprintf("%s are required", strings.ReplaceAll(props, "'", ""))
	}

	path = "root"
	if len(contexts) > 0 {
		path = strings.Join(contexts, " > ")
	}
	return path, finalMsg
}
```

**Step 2: Vérifier que ça compile**

```bash
go build ./internal/validator/...
```
Expected: pas d'erreur.

**Step 3: Commit**

```bash
git add internal/validator/
git commit -m "feat: add validator package with worker pool"
```

---

### Task 7: internal/reporter — Interface + Reporter terminal

**Files:**
- Create: `internal/reporter/reporter.go`
- Create: `internal/reporter/terminal.go`

**Step 1: Implémenter l'interface**

```go
// internal/reporter/reporter.go
package reporter

import (
	"jsnsch/internal/validator"
	"time"
)

type Report struct {
	Duration time.Duration
	Results  []validator.TypeResult
}

type Reporter interface {
	Render(r Report) error
}
```

**Step 2: Implémenter le reporter terminal**

```go
// internal/reporter/terminal.go
package reporter

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

var separator = strings.Repeat("-", 100)

type Terminal struct {
	Verbose bool
}

func (t *Terminal) Render(r Report) error {
	fmt.Printf("\n%s\n", separator)

	for _, res := range r.Results {
		for _, d := range res.Details {
			fmt.Printf("%s %s : %s : %s\n",
				color.RedString("❌"),
				color.YellowString(d.File),
				d.Path,
				d.Message,
			)
		}
	}

	fmt.Printf("\nSummary:\n")
	var totalFiles, totalErrors int
	for _, res := range r.Results {
		totalFiles += res.Success + res.Errors
		totalErrors += res.Errors
		prefix := color.GreenString(">")
		errStr := fmt.Sprintf("%d Erreurs", res.Errors)
		if res.Errors > 0 {
			prefix = color.RedString(">")
			errStr = color.RedString(errStr)
		}
		fmt.Printf("%s Nodes %-2s : %s | %s\n",
			prefix, res.Type,
			color.GreenString("%4d OK", res.Success),
			errStr,
		)
	}

	fmt.Printf("\n%s\n\n", separator)
	finalMsg := fmt.Sprintf("TOTAL : %d fichiers analysés | %d erreurs", totalFiles, totalErrors)
	fmt.Printf("⏱️  Temps total : %v\n", r.Duration.Round(1000000))

	if totalErrors == 0 {
		color.Green("⭐ " + finalMsg + " (CONFORME)")
	} else {
		color.Red("🚨 " + finalMsg + " (NON CONFORME)")
	}
	return nil
}
```

**Step 3: Vérifier que ça compile**

```bash
go build ./internal/reporter/...
```
Expected: pas d'erreur.

**Step 4: Commit**

```bash
git add internal/reporter/reporter.go internal/reporter/terminal.go
git commit -m "feat: add reporter interface and terminal implementation"
```

---

### Task 8: internal/reporter — JSON reporter

**Files:**
- Create: `internal/reporter/json.go`
- Create: `internal/reporter/json_test.go`

**Step 1: Écrire le test**

```go
// internal/reporter/json_test.go
package reporter_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"jsnsch/internal/reporter"
	"jsnsch/internal/validator"
)

func TestJSONReporter(t *testing.T) {
	var buf bytes.Buffer
	r := &reporter.JSON{Writer: &buf}

	report := reporter.Report{
		Duration: 500 * time.Millisecond,
		Results: []validator.TypeResult{
			{Type: "M", Success: 10, Errors: 1, Details: []validator.FileError{
				{File: "a_M.json", Path: "root", Message: "x is required"},
			}},
		},
	}
	if err := r.Render(report); err != nil {
		t.Fatal(err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if out["total"].(float64) != 11 {
		t.Errorf("expected total=11, got %v", out["total"])
	}
	if out["errors"].(float64) != 1 {
		t.Errorf("expected errors=1, got %v", out["errors"])
	}
	if out["success"].(bool) != false {
		t.Error("expected success=false")
	}
}
```

**Step 2: Vérifier que le test échoue**

```bash
go test ./internal/reporter/... -run TestJSONReporter
```
Expected: FAIL.

**Step 3: Implémenter**

```go
// internal/reporter/json.go
package reporter

import (
	"encoding/json"
	"io"
	"os"
)

type jsonOutput struct {
	DurationMs int64                `json:"duration_ms"`
	Total      int                  `json:"total"`
	Errors     int                  `json:"errors"`
	Success    bool                 `json:"success"`
	Results    []jsonTypeResult     `json:"results"`
}

type jsonTypeResult struct {
	Type    string        `json:"type"`
	Success int           `json:"success"`
	Errors  int           `json:"errors"`
	Details []jsonDetail  `json:"details,omitempty"`
}

type jsonDetail struct {
	File    string `json:"file"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

type JSON struct {
	Writer io.Writer
}

func (j *JSON) Render(r Report) error {
	w := j.Writer
	if w == nil {
		w = os.Stdout
	}

	var total, errs int
	results := make([]jsonTypeResult, 0, len(r.Results))
	for _, res := range r.Results {
		total += res.Success + res.Errors
		errs += res.Errors
		tr := jsonTypeResult{
			Type:    res.Type,
			Success: res.Success,
			Errors:  res.Errors,
		}
		for _, d := range res.Details {
			tr.Details = append(tr.Details, jsonDetail{File: d.File, Path: d.Path, Message: d.Message})
		}
		results = append(results, tr)
	}

	out := jsonOutput{
		DurationMs: r.Duration.Milliseconds(),
		Total:      total,
		Errors:     errs,
		Success:    errs == 0,
		Results:    results,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
```

**Step 4: Vérifier que les tests passent**

```bash
go test ./internal/reporter/... -v
```
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/reporter/json.go internal/reporter/json_test.go
git commit -m "feat: add JSON reporter"
```

---

### Task 9: internal/reporter — JUnit reporter

**Files:**
- Create: `internal/reporter/junit.go`
- Create: `internal/reporter/junit_test.go`

**Step 1: Écrire le test**

```go
// internal/reporter/junit_test.go
package reporter_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"jsnsch/internal/reporter"
	"jsnsch/internal/validator"
)

func TestJUnitReporter(t *testing.T) {
	var buf bytes.Buffer
	r := &reporter.JUnit{Writer: &buf}

	report := reporter.Report{
		Duration: 200 * time.Millisecond,
		Results: []validator.TypeResult{
			{Type: "M", Success: 5, Errors: 1, Details: []validator.FileError{
				{File: "bad_M.json", Path: "root > address", Message: "street is required"},
			}},
		},
	}
	if err := r.Render(report); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "<testsuites") {
		t.Error("expected <testsuites> element")
	}
	if !strings.Contains(out, "bad_M.json") {
		t.Error("expected failing test case for bad_M.json")
	}
	if !strings.Contains(out, "street is required") {
		t.Error("expected failure message")
	}
}
```

**Step 2: Vérifier que le test échoue**

```bash
go test ./internal/reporter/... -run TestJUnitReporter
```
Expected: FAIL.

**Step 3: Implémenter**

```go
// internal/reporter/junit.go
package reporter

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
)

type junitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	TestSuites []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	TestCases []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name    string         `xml:"name,attr"`
	Failure *junitFailure  `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

type JUnit struct {
	Writer io.Writer
}

func (j *JUnit) Render(r Report) error {
	w := j.Writer
	if w == nil {
		w = os.Stdout
	}

	suites := junitTestSuites{}
	for _, res := range r.Results {
		suite := junitTestSuite{
			Name:     fmt.Sprintf("Type %s", res.Type),
			Tests:    res.Success + res.Errors,
			Failures: res.Errors,
		}
		// Add passing tests (no failure element)
		for i := 0; i < res.Success; i++ {
			suite.TestCases = append(suite.TestCases, junitTestCase{
				Name: fmt.Sprintf("valid_file_%d", i+1),
			})
		}
		// Add failing tests
		for _, d := range res.Details {
			suite.TestCases = append(suite.TestCases, junitTestCase{
				Name: d.File,
				Failure: &junitFailure{
					Message: d.Path,
					Text:    d.Message,
				},
			})
		}
		suites.TestSuites = append(suites.TestSuites, suite)
	}

	fmt.Fprintln(w, xml.Header)
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(suites); err != nil {
		return err
	}
	return enc.Flush()
}
```

**Step 4: Vérifier que les tests passent**

```bash
go test ./internal/reporter/... -v
```
Expected: PASS (les trois tests reporter).

**Step 5: Commit**

```bash
git add internal/reporter/junit.go internal/reporter/junit_test.go
git commit -m "feat: add JUnit XML reporter"
```

---

### Task 10: cmd/root.go — Cobra root + Viper init

**Files:**
- Create: `cmd/root.go`

**Step 1: Implémenter**

```go
// cmd/root.go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "jsnsch",
	Short: "Validateur JSON Schema multithreadé",
	Long: `jsnsch valide des fichiers JSON contre leurs schémas associés.

Les fichiers doivent suivre la convention *_<TYPE>.json.
Les schémas doivent être nommés json-schema-Node_<TYPE>.json.

Exemples:
  jsnsch validate ./data --all
  jsnsch validate ./data --types M,R,I --output json
  jsnsch schema list --schemas ./schemas
  jsnsch config init`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "fichier de config (défaut: .jsnsch.yaml)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, _ := os.UserHomeDir()
		viper.AddConfigPath(".")
		viper.AddConfigPath(filepath.Join(home, ".config", "jsnsch"))
		viper.SetConfigName(".jsnsch")
		viper.SetConfigType("yaml")
	}
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()
}
```

**Step 2: Vérifier que ça compile**

```bash
go build ./cmd/...
```
Expected: pas d'erreur.

**Step 3: Commit**

```bash
git add cmd/root.go
git commit -m "feat: add cobra root command with viper config init"
```

---

### Task 11: cmd/validate.go — Commande validate avec barres de progression

**Files:**
- Create: `cmd/validate.go`

**Step 1: Implémenter**

```go
// cmd/validate.go
package cmd

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"

	"jsnsch/internal/reporter"
	"jsnsch/internal/scanner"
	"jsnsch/internal/schema"
	"jsnsch/internal/validator"
)

var validateCmd = &cobra.Command{
	Use:   "validate <directory>",
	Short: "Valide les fichiers JSON d'un dossier contre leurs schémas",
	Long: `Parcourt <directory> récursivement et valide chaque fichier *_<TYPE>.json
contre le schéma json-schema-Node_<TYPE>.json correspondant.

Exemples:
  jsnsch validate ./data --all
  jsnsch validate ./data --types M,R --verbose
  jsnsch validate ./data --all --output json > results.json
  jsnsch validate ./data --all --output junit > results.xml`,
	Args: cobra.ExactArgs(1),
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)

	f := validateCmd.Flags()
	f.String("schemas", ".", "Dossier contenant les schémas JSON")
	f.StringSlice("types", nil, "Types à valider (ex: M,R,I). Défaut: auto-détecté")
	f.Bool("all", false, "Valider tous les types détectés")
	f.String("output", "terminal", "Format de sortie: terminal | json | junit")
	f.Bool("verbose", false, "Afficher le détail complet des erreurs")
	f.Int("workers", 0, "Nombre de workers (0 = NumCPU)")
	f.Bool("no-progress", false, "Désactiver les barres de progression")

	_ = viper.BindPFlags(f)
}

func runValidate(cmd *cobra.Command, args []string) error {
	dir := args[0]
	schemasDir := viper.GetString("schemas")
	typesFlag := viper.GetStringSlice("types")
	allFlag := viper.GetBool("all")
	outputFmt := viper.GetString("output")
	verbose := viper.GetBool("verbose")
	workers := viper.GetInt("workers")
	noProgress := viper.GetBool("no-progress")

	// Resolve types
	var types []string
	if allFlag || len(typesFlag) == 0 {
		detected, err := schema.DetectTypes(schemasDir)
		if err != nil {
			return fmt.Errorf("détection des types: %w", err)
		}
		if len(typesFlag) > 0 && !allFlag {
			types = typesFlag
		} else {
			types = detected
		}
	} else {
		types = typesFlag
	}

	if len(types) == 0 {
		return fmt.Errorf("aucun type trouvé dans %s — vérifiez --schemas ou utilisez --types", schemasDir)
	}

	if outputFmt == "terminal" {
		fmt.Printf("\n🚀 Analyse de : %s\n", color.CyanString(dir))
		fmt.Printf("📂 Schémas   : %s\n", color.CyanString(schemasDir))
		fmt.Printf("🏷️  Types     : %v\n\n", types)
	}

	// Scan files
	filesByType, err := scanner.ScanFiles(dir, types)
	if err != nil {
		return fmt.Errorf("scan du dossier: %w", err)
	}

	totalTasks := 0
	for _, files := range filesByType {
		totalTasks += len(files)
	}
	if totalTasks == 0 {
		color.Yellow("⚠️  Aucun fichier trouvé pour les types demandés.")
		return nil
	}

	// Sort types for consistent display
	typeOrder := make(map[string]int, len(types))
	for i, t := range types {
		typeOrder[t] = i
	}

	// Setup progress bars
	var p *mpb.Progress
	bars := make(map[string]*mpb.Bar)
	if outputFmt == "terminal" && !noProgress {
		p = mpb.New(mpb.WithWidth(60))
		for _, t := range types {
			files := filesByType[t]
			if len(files) == 0 {
				continue
			}
			name := fmt.Sprintf("🔍 [Type %s]", t)
			bars[t] = p.AddBar(int64(len(files)),
				mpb.PrependDecorators(
					decor.Name(name, decor.WC{W: runewidth.StringWidth(name) + 1}),
					decor.CountersNoUnit("%d/%d", decor.WC{W: 10}),
				),
				mpb.AppendDecorators(
					decor.Percentage(decor.WC{W: 5}),
					decor.OnComplete(decor.Name(""), color.GreenString("  ✅")),
				),
			)
		}
	}

	numWorkers := workers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	if outputFmt == "terminal" {
		fmt.Printf("👷 Workers actifs : %d\n", numWorkers)
	}

	start := time.Now()
	results := validator.Run(filesByType, schema.NewLocalLoader(schemasDir), validator.Options{
		Workers: numWorkers,
		Verbose: verbose,
		OnProgress: func(typeNode string) {
			if b, ok := bars[typeNode]; ok {
				b.Increment()
			}
		},
	})

	if p != nil {
		p.Wait()
	}
	duration := time.Since(start)

	// Sort results
	sort.Slice(results, func(i, j int) bool {
		return typeOrder[results[i].Type] < typeOrder[results[j].Type]
	})

	// Render
	report := reporter.Report{Duration: duration, Results: results}
	var r reporter.Reporter
	switch outputFmt {
	case "json":
		r = &reporter.JSON{}
	case "junit":
		r = &reporter.JUnit{}
	default:
		r = &reporter.Terminal{Verbose: verbose}
	}

	if err := r.Render(report); err != nil {
		return err
	}

	// Exit code
	for _, res := range results {
		if res.Errors > 0 {
			os.Exit(1)
		}
	}
	return nil
}
```

**Step 2: Vérifier que ça compile**

```bash
go build ./cmd/...
```
Expected: pas d'erreur.

**Step 3: Commit**

```bash
git add cmd/validate.go
git commit -m "feat: add validate command with progress bars and multi-format output"
```

---

### Task 12: cmd/schema.go — Commandes schema list/check

**Files:**
- Create: `cmd/schema.go`

**Step 1: Implémenter**

```go
// cmd/schema.go
package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"jsnsch/internal/schema"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Gérer et inspecter les schémas JSON",
}

var schemaListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lister les schémas détectés dans le dossier --schemas",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("schemas")
		types, err := schema.DetectTypes(dir)
		if err != nil {
			return fmt.Errorf("lecture du dossier schémas: %w", err)
		}
		if len(types) == 0 {
			color.Yellow("Aucun schéma trouvé dans %s", dir)
			return nil
		}
		fmt.Printf("Schémas détectés dans %s:\n", color.CyanString(dir))
		for _, t := range types {
			fmt.Printf("  %s Type %s → json-schema-Node_%s.json\n", color.GreenString("✓"), t, t)
		}
		return nil
	},
}

var schemaCheckCmd = &cobra.Command{
	Use:   "check <type>",
	Short: "Vérifier qu'un schéma est valide et chargeable",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("schemas")
		typeNode := args[0]
		loader := schema.NewLocalLoader(dir)
		_, err := loader.Load(typeNode)
		if err != nil {
			color.Red("❌ Schéma invalide pour le type %s: %v", typeNode, err)
			return fmt.Errorf("schéma invalide: %w", err)
		}
		color.Green("✅ Schéma pour le type %s : OK", typeNode)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(schemaCmd)
	schemaCmd.AddCommand(schemaListCmd)
	schemaCmd.AddCommand(schemaCheckCmd)

	schemaListCmd.Flags().String("schemas", ".", "Dossier contenant les schémas")
	schemaCheckCmd.Flags().String("schemas", ".", "Dossier contenant les schémas")
}
```

**Step 2: Vérifier que ça compile**

```bash
go build ./cmd/...
```
Expected: pas d'erreur.

**Step 3: Commit**

```bash
git add cmd/schema.go
git commit -m "feat: add schema list and check subcommands"
```

---

### Task 13: cmd/config.go — Commandes config init/show

**Files:**
- Create: `cmd/config.go`

**Step 1: Implémenter**

```go
// cmd/config.go
package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Gérer la configuration de jsnsch",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Générer un fichier .jsnsch.yaml exemple dans le dossier courant",
	RunE: func(cmd *cobra.Command, args []string) error {
		const template = `# jsnsch configuration
# Documentation: jsnsch --help

# Dossier contenant les schémas JSON (json-schema-Node_<TYPE>.json)
schemas: .

# Types à valider. Si vide, auto-détecté depuis le dossier schemas.
# types:
#   - M
#   - R
#   - I

# Format de sortie: terminal | json | junit
output: terminal

# Afficher le détail complet des erreurs de validation
verbose: false

# Nombre de workers parallèles (0 = NumCPU automatique)
workers: 0

# Désactiver les barres de progression (utile en CI/CD)
no_progress: false
`
		const filename = ".jsnsch.yaml"
		if _, err := os.Stat(filename); err == nil {
			color.Yellow("⚠️  %s existe déjà. Supprimez-le avant de relancer init.", filename)
			return nil
		}
		if err := os.WriteFile(filename, []byte(template), 0644); err != nil {
			return fmt.Errorf("impossible de créer %s: %w", filename, err)
		}
		color.Green("✅ %s créé avec succès.", filename)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Afficher la configuration active (fusion CLI + fichier + défauts)",
	Run: func(cmd *cobra.Command, args []string) {
		cfgUsed := viper.ConfigFileUsed()
		if cfgUsed == "" {
			cfgUsed = "(aucun fichier de config trouvé)"
		}
		fmt.Printf("Fichier de config : %s\n\n", color.CyanString(cfgUsed))
		fmt.Printf("  schemas     : %s\n", viper.GetString("schemas"))
		fmt.Printf("  types       : %v\n", viper.GetStringSlice("types"))
		fmt.Printf("  output      : %s\n", viper.GetString("output"))
		fmt.Printf("  verbose     : %v\n", viper.GetBool("verbose"))
		fmt.Printf("  workers     : %d\n", viper.GetInt("workers"))
		fmt.Printf("  no-progress : %v\n", viper.GetBool("no-progress"))
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
}
```

**Step 2: Vérifier que ça compile**

```bash
go build ./cmd/...
```
Expected: pas d'erreur.

**Step 3: Commit**

```bash
git add cmd/config.go
git commit -m "feat: add config init and show subcommands"
```

---

### Task 14: main.go — Point d'entrée minimal + build final

**Files:**
- Modify: `main.go` (remplacer tout le contenu)

**Step 1: Remplacer main.go**

```go
// main.go
package main

import "jsnsch/cmd"

func main() {
	cmd.Execute()
}
```

**Step 2: Build Linux et Windows**

```bash
go build -o jsnsch .
GOOS=windows GOARCH=amd64 go build -o jsnsch.exe .
```
Expected: deux binaires créés sans erreur.

**Step 3: Vérifier --help**

```bash
./jsnsch --help
./jsnsch validate --help
./jsnsch schema --help
./jsnsch config --help
```
Expected: aide affichée pour chaque commande.

**Step 4: Lancer tous les tests**

```bash
go test ./... -v
```
Expected: PASS.

**Step 5: go vet**

```bash
go vet ./...
```
Expected: pas d'erreur.

**Step 6: Commit final**

```bash
git add main.go
git commit -m "feat: wire cobra CLI entry point, replace main.go"
```

---

## Résumé des commits attendus

1. `chore: add cobra and viper dependencies`
2. `feat: add config package with viper loading`
3. `feat: add schema loader interface and local implementation`
4. `feat: add automatic type detection from schema directory`
5. `feat: add file scanner package`
6. `feat: add validator package with worker pool`
7. `feat: add reporter interface and terminal implementation`
8. `feat: add JSON reporter`
9. `feat: add JUnit XML reporter`
10. `feat: add cobra root command with viper config init`
11. `feat: add validate command with progress bars and multi-format output`
12. `feat: add schema list and check subcommands`
13. `feat: add config init and show subcommands`
14. `feat: wire cobra CLI entry point, replace main.go`
