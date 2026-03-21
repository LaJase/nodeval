package schema_test

import (
	"os"
	"path/filepath"
	"testing"

	"nodeval/internal/schema"
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
