package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestRootCmd_NoUsageOnValidationError verifies that when a subcommand returns
// a ValidationError, the usage text is not printed to stderr.
func TestRootCmd_NoUsageOnValidationError(t *testing.T) {
	out := &bytes.Buffer{}

	root := &cobra.Command{Use: "nodeval", SilenceErrors: true}
	root.SilenceUsage = rootCmd.SilenceUsage

	child := &cobra.Command{
		Use:  "validate",
		RunE: func(cmd *cobra.Command, args []string) error {
			return &ValidationError{Msg: "1 invalid file(s)"}
		},
	}
	root.AddCommand(child)
	root.SetOut(out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"validate"})
	_ = root.Execute()

	if strings.Contains(out.String(), "Usage:") {
		t.Errorf("usage must not be shown on ValidationError, got output:\n%s", out.String())
	}
}
