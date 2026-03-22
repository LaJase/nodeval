package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"nodeval/internal/config"
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

func TestDefaultSchemaPattern(t *testing.T) {
	cfg := config.Default()
	if cfg.SchemaPattern != "json-schema-Node_{type}.json" {
		t.Errorf("expected default schema pattern, got %q", cfg.SchemaPattern)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	content := []byte("output: json\nworkers: 4\nverbose: true\n")
	_ = os.WriteFile(filepath.Join(dir, ".nodeval.yaml"), content, 0o644)

	cfg, err := config.LoadFrom(filepath.Join(dir, ".nodeval.yaml"))
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
