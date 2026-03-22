package cmd

import (
	"path/filepath"
	"testing"
)

func TestConfigUnset_RemovesKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	_ = writeConfigFile(path, map[string]any{"schemas": "./data", "output": "json"})

	if err := runConfigUnset(path, "output"); err != nil {
		t.Fatal(err)
	}

	m, _ := readConfigFile(path)
	if _, ok := m["output"]; ok {
		t.Error("expected output to be removed")
	}
	if m["schemas"] != "./data" {
		t.Error("expected schemas to remain")
	}
}

func TestConfigUnset_AbsentKey_NoError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	_ = writeConfigFile(path, map[string]any{"schemas": "."})

	if err := runConfigUnset(path, "output"); err != nil {
		t.Errorf("expected no error for absent key, got: %v", err)
	}
	m, _ := readConfigFile(path)
	if m["schemas"] != "." {
		t.Errorf("expected file to be unchanged, got schemas=%v", m["schemas"])
	}
}

func TestConfigUnset_MissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	if err := runConfigUnset(path, "schemas"); err == nil {
		t.Error("expected error when file does not exist")
	}
}

func TestConfigUnset_UnknownKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	_ = writeConfigFile(path, map[string]any{"schemas": "."})
	if err := runConfigUnset(path, "badkey"); err == nil {
		t.Error("expected error for unknown key")
	}
}
