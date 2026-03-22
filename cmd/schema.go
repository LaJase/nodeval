// cmd/schema.go
package cmd

import (
	"fmt"

	"nodeval/internal/schema"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Manage and inspect JSON schemas",
}

var schemaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List schemas detected in the --schemas directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("schemas")
		pattern := resolveSchemaPattern(cmd)

		types, err := schema.DetectTypes(dir, pattern)
		if err != nil {
			return fmt.Errorf("reading schemas directory: %w", err)
		}
		if len(types) == 0 {
			color.Yellow("No schemas found in %s", dir)
			return nil
		}
		fmt.Printf("Schemas detected in %s:\n", color.CyanString(dir))
		prefix, suffix, _ := schema.ParsePattern(pattern)
		for _, t := range types {
			filename := prefix + t + suffix
			fmt.Printf("  %s Type %s → %s\n", color.GreenString("✓"), t, filename)
		}
		return nil
	},
}

var schemaCheckCmd = &cobra.Command{
	Use:   "check [type...]",
	Short: "Check that one or more schemas are valid and loadable",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("schemas")
		pattern := resolveSchemaPattern(cmd)
		all, _ := cmd.Flags().GetBool("all")

		var types []string
		if all {
			detected, err := schema.DetectTypes(dir, pattern)
			if err != nil {
				return fmt.Errorf("reading schemas directory: %w", err)
			}
			if len(detected) == 0 {
				color.Yellow("No schemas found in %s", dir)
				return nil
			}
			types = detected
		} else {
			if len(args) == 0 {
				return fmt.Errorf("specify at least one type or use --all")
			}
			types = args
		}

		loader, err := schema.NewLocalLoader(dir, pattern)
		if err != nil {
			return &ConfigError{Msg: fmt.Sprintf("invalid schema_pattern: %v", err)}
		}

		var failed int
		for _, t := range types {
			_, err = loader.Load(t)
			if err != nil {
				color.Red("❌ Type %s: %v", t, err)
				failed++
			} else {
				color.Green("✅ Type %s: OK", t)
			}
		}
		if failed > 0 {
			return &ConfigError{Msg: fmt.Sprintf("%d invalid schema(s)", failed)}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(schemaCmd)
	schemaCmd.AddCommand(schemaListCmd)
	schemaCmd.AddCommand(schemaCheckCmd)

	// PersistentFlags inherited by list and check
	schemaCmd.PersistentFlags().String("schemas", ".", "Directory containing schemas")
	schemaCmd.PersistentFlags().String("schema-pattern", "", "Schema filename pattern (e.g. schema_{type}.json)")
	schemaCheckCmd.Flags().Bool("all", false, "Check all auto-detected types")
}
