package validator_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"nodeval/internal/validator"
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
	_ = os.WriteFile(path, []byte(`{"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}`), 0644)
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
	_ = os.WriteFile(p, b, 0644)
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

func TestRun_MixedValidInvalidFiles(t *testing.T) {
	dir := t.TempDir()
	sch := makeSchema(t)
	loader := &stubLoader{schema: sch}

	var files []string
	for i := 0; i < 30; i++ {
		files = append(files, writeJSON(t, dir, fmt.Sprintf("ok_%d_T.json", i), map[string]int{"id": i}))
	}
	for i := 0; i < 20; i++ {
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
