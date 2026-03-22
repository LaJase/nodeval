package cmd

import (
	"path/filepath"
	"testing"
)

func TestConfigUnset_RemovesKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	_ = writeConfigFile(path, map[string]any{"schemas": "./data", "output": "json"})

	removed, err := runConfigUnset(path, "output")
	if err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Error("expected key to be removed")
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

	removed, err := runConfigUnset(path, "output")
	if err != nil {
		t.Errorf("expected no error for absent key, got: %v", err)
	}
	if removed {
		t.Error("expected removed=false for absent key")
	}
	m, _ := readConfigFile(path)
	if m["schemas"] != "." {
		t.Errorf("expected file to be unchanged, got schemas=%v", m["schemas"])
	}
}

func TestConfigUnset_MissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	_, err := runConfigUnset(path, "schemas")
	if err == nil {
		t.Error("expected error when file does not exist")
	}
}

func TestConfigUnset_UnknownKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	_ = writeConfigFile(path, map[string]any{"schemas": "."})
	_, err := runConfigUnset(path, "badkey")
	if err == nil {
		t.Error("expected error for unknown key")
	}
}
