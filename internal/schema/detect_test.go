package schema_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"nodeval/internal/schema"
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
