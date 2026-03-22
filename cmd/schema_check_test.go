package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestSchemaCheck_ReadsSchemasDirFromViper(t *testing.T) {
	dir := t.TempDir()
	writeSchema(t, dir, "M")
	if err := execSchemaCheck(dir, []string{"--all"}); err != nil {
		t.Fatalf("expected success when schemas dir set in viper, got: %v", err)
	}
}

func writeSchema(t *testing.T, dir, typ string) {
	t.Helper()
	name := "json-schema-Node_" + typ + ".json"
	err := os.WriteFile(filepath.Join(dir, name), []byte(`{"type": "object"}`), 0o644)
	if err != nil {
		t.Fatal(err)
	}
}

// execSchemaCheck builds a minimal fresh Cobra tree per call to prevent flag
// state from leaking between tests. It reuses only schemaCheckCmd.RunE; the
// Use/Args fields are re-declared here and are not kept in sync automatically.
//
// dir is injected via viper at the override level so it survives the
// initConfig SetDefault call triggered by cobra.OnInitialize.
func execSchemaCheck(dir string, args []string) error {
	viper.Reset()
	viper.Set("schemas", dir)
	viper.Set("schema_pattern", "json-schema-Node_{type}.json")

	root := &cobra.Command{Use: "nodeval"}
	parent := &cobra.Command{Use: "schema"}
	parent.PersistentFlags().String("schemas", "", "")
	parent.PersistentFlags().String("schema-pattern", "", "")

	child := &cobra.Command{
		Use:  "check [type...]",
		Args: cobra.ArbitraryArgs,
		RunE: schemaCheckCmd.RunE,
	}
	child.Flags().Bool("all", false, "")

	root.AddCommand(parent)
	parent.AddCommand(child)
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))

	cmdArgs := append([]string{"schema", "check"}, args...)
	root.SetArgs(cmdArgs)
	return root.Execute()
}

func TestSchemaCheck_SingleValid(t *testing.T) {
	dir := t.TempDir()
	writeSchema(t, dir, "M")
	if err := execSchemaCheck(dir, []string{"M"}); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
}

func TestSchemaCheck_MultipleValid(t *testing.T) {
	dir := t.TempDir()
	writeSchema(t, dir, "M")
	writeSchema(t, dir, "R")
	if err := execSchemaCheck(dir, []string{"M", "R"}); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
}

func TestSchemaCheck_AllFlag(t *testing.T) {
	dir := t.TempDir()
	writeSchema(t, dir, "M")
	writeSchema(t, dir, "R")
	if err := execSchemaCheck(dir, []string{"--all"}); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
}

func TestSchemaCheck_Missing(t *testing.T) {
	dir := t.TempDir()
	if err := execSchemaCheck(dir, []string{"M"}); err == nil {
		t.Fatal("expected error for missing schema")
	}
}

func TestSchemaCheck_NoArgsNoAll(t *testing.T) {
	dir := t.TempDir()
	err := execSchemaCheck(dir, []string{})
	if err == nil {
		t.Fatal("expected error when no args and no --all")
	}
	if !strings.Contains(err.Error(), "specify at least one type or use --all") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestSchemaCheck_AllEmptyDir(t *testing.T) {
	dir := t.TempDir()
	if err := execSchemaCheck(dir, []string{"--all"}); err != nil {
		t.Fatalf("expected success for empty dir with --all, got: %v", err)
	}
}
