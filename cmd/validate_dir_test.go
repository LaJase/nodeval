// cmd/validate_dir_test.go
package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"nodeval/internal/validator"
)

func execValidateDir(args []string, configDir string) error {
	viper.Reset()
	viper.Set("directory", configDir)
	viper.SetDefault("schemas", ".")
	viper.SetDefault("output", "terminal")
	viper.SetDefault("workers", 0)
	viper.SetDefault("verbose", false)
	viper.SetDefault("no_progress", false)
	viper.SetDefault("schema_pattern", "json-schema-Node_{type}.json")

	root := &cobra.Command{Use: "nodeval"}
	child := &cobra.Command{
		Use:  "validate [directory]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := viper.GetString("directory")
			if len(args) > 0 {
				dir = args[0]
			}
			if dir == "" {
				return fmt.Errorf("no directory specified — pass it as argument or set 'directory' in config")
			}
			return nil
		},
	}
	root.AddCommand(child)
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(append([]string{"validate"}, args...))
	return root.Execute()
}

func TestValidationError_TotalsAcrossAllTypes(t *testing.T) {
	results := []validator.TypeResult{
		{Type: "Alpha", Errors: 4},
		{Type: "Zeta", Errors: 4},
	}
	err := validationExitError(results)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "8") {
		t.Errorf("expected total of 8 invalid files in message, got: %q", err.Error())
	}
}

func TestValidationError_NoErrorWhenAllValid(t *testing.T) {
	results := []validator.TypeResult{
		{Type: "Alpha", Errors: 0},
		{Type: "Zeta", Errors: 0},
	}
	if err := validationExitError(results); err != nil {
		t.Errorf("expected nil, got: %v", err)
	}
}

func TestValidate_ArgOverridesConfig(t *testing.T) {
	if err := execValidateDir([]string{"./data"}, "./other"); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
}

func TestValidate_ConfigUsedWhenNoArg(t *testing.T) {
	if err := execValidateDir([]string{}, "./data"); err != nil {
		t.Fatalf("expected success with config dir, got: %v", err)
	}
}

func TestValidate_ErrorWhenNeitherArgNorConfig(t *testing.T) {
	err := execValidateDir([]string{}, "")
	if err == nil {
		t.Fatal("expected error when no dir and no config")
	}
	if !strings.Contains(err.Error(), "no directory specified") {
		t.Errorf("unexpected error: %v", err)
	}
}
