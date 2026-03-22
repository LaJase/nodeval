package cmd

import (
	"path/filepath"
	"testing"
)

func TestReadWriteConfigFile_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")

	data := map[string]any{"schemas": "./schemas", "workers": 4}
	if err := writeConfigFile(path, data); err != nil {
		t.Fatal(err)
	}

	got, err := readConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["schemas"] != "./schemas" {
		t.Errorf("expected schemas=./schemas, got %v", got["schemas"])
	}
}

func TestReadConfigFile_Missing(t *testing.T) {
	got, err := readConfigFile("/nonexistent/path/.nodeval.yaml")
	if err != nil {
		t.Fatal("expected empty map, not error")
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
	_ = got
}

func TestValidateKey(t *testing.T) {
	if err := validateKey("schemas"); err != nil {
		t.Errorf("expected schemas to be valid: %v", err)
	}
	if err := validateKey("unknown_key"); err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestCoerceValue(t *testing.T) {
	v, err := coerceValue("workers", "4")
	if err != nil || v != 4 {
		t.Errorf("expected int 4, got %v err %v", v, err)
	}
	v, err = coerceValue("verbose", "true")
	if err != nil || v != true {
		t.Errorf("expected bool true, got %v err %v", v, err)
	}
	v, err = coerceValue("schemas", "./data")
	if err != nil || v != "./data" {
		t.Errorf("expected string ./data, got %v err %v", v, err)
	}
}
