package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestConfigSet_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")

	if err := runConfigSet(path, "schemas", "./data"); err != nil {
		t.Fatal(err)
	}

	m, _ := readConfigFile(path)
	if m["schemas"] != "./data" {
		t.Errorf("expected schemas=./data, got %v", m["schemas"])
	}
}

func TestConfigSet_OverwritesValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	_ = writeConfigFile(path, map[string]any{"schemas": "."})

	if err := runConfigSet(path, "schemas", "./new"); err != nil {
		t.Fatal(err)
	}

	m, _ := readConfigFile(path)
	if m["schemas"] != "./new" {
		t.Errorf("expected schemas=./new, got %v", m["schemas"])
	}
}

func TestConfigSet_TypeCoercion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")

	if err := runConfigSet(path, "workers", "8"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	var m map[string]any
	_ = yaml.Unmarshal(data, &m)
	if m["workers"] != 8 {
		t.Errorf("expected int 8, got %T %v", m["workers"], m["workers"])
	}
}

func TestConfigSet_UnknownKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	if err := runConfigSet(path, "badkey", "value"); err == nil {
		t.Error("expected error for unknown key")
	}
}
