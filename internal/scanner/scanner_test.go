package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"nodeval/internal/scanner"
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

func TestScanFiles_MultipleUnderscores(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		"foo_bar_baz_M.json", // multiple underscores — type is last segment
		"foo_bar_MR.json",    // type "MR" should not match type "M"
		"_M.json",            // leading underscore
	}
	for _, f := range files {
		_ = os.WriteFile(filepath.Join(dir, f), []byte(`{}`), 0644)
	}

	result, err := scanner.ScanFiles(dir, []string{"M", "MR"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result["M"]) != 2 {
		t.Errorf("expected 2 M files, got %d: %v", len(result["M"]), result["M"])
	}
	if len(result["MR"]) != 1 {
		t.Errorf("expected 1 MR file, got %d", len(result["MR"]))
	}
}
