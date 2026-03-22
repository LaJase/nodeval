package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func execConfigGet(key string, cfgPath ...string) (string, error) {
	viper.Reset()
	viper.SetDefault("schemas", ".")
	viper.SetDefault("output", "terminal")
	viper.SetDefault("workers", 0)
	viper.SetDefault("verbose", false)
	viper.SetDefault("no_progress", false)
	viper.SetDefault("schema_pattern", "json-schema-Node_{type}.json")

	if len(cfgPath) > 0 && cfgPath[0] != "" {
		// Set the package-level cfgFile var so that initConfig (triggered by
		// cobra.OnInitialize on Execute) uses this path via viper.SetConfigFile.
		// We cannot use viper.SetConfigFile here directly because initConfig's
		// viper.SetConfigName call would clear it before ReadInConfig runs.
		cfgFile = cfgPath[0]
		defer func() { cfgFile = "" }()
	}

	buf := &bytes.Buffer{}
	root := &cobra.Command{Use: "nodeval"}
	parent := &cobra.Command{Use: "config"}
	child := &cobra.Command{
		Use:  "get <key>",
		Args: cobra.ExactArgs(1),
		RunE: configGetCmd.RunE,
	}
	root.AddCommand(parent)
	parent.AddCommand(child)
	root.SetOut(buf)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"config", "get", key})
	err := root.Execute()
	return buf.String(), err
}

func TestConfigGet_Default(t *testing.T) {
	out, err := execConfigGet("output")
	if err != nil {
		t.Fatal(err)
	}
	if out != "terminal\n" {
		t.Errorf("expected 'terminal\\n', got %q", out)
	}
}

func TestConfigGet_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	_ = writeConfigFile(path, map[string]any{"output": "json"})

	out, err := execConfigGet("output", path)
	if err != nil {
		t.Fatal(err)
	}
	if out != "json\n" {
		t.Errorf("expected 'json\\n', got %q", out)
	}
}

func TestConfigGet_UnknownKey(t *testing.T) {
	_, err := execConfigGet("badkey")
	if err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestConfigGet_BoolKey(t *testing.T) {
	out, err := execConfigGet("verbose")
	if err != nil {
		t.Fatal(err)
	}
	if out != "false\n" {
		t.Errorf("expected 'false\\n', got %q", out)
	}
}

func TestConfigGet_IntKey(t *testing.T) {
	out, err := execConfigGet("workers")
	if err != nil {
		t.Fatal(err)
	}
	if out != "0\n" {
		t.Errorf("expected '0\\n', got %q", out)
	}
}
