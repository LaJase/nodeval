// cmd/schema.go
package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"nodeval/internal/schema"
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
		pattern, _ := cmd.Flags().GetString("schema-pattern")
		if pattern == "" {
			pattern = viper.GetString("schema_pattern")
		}

		types, err := schema.DetectTypes(dir, pattern)
		if err != nil {
			return fmt.Errorf("reading schemas directory: %w", err)
		}
		if len(types) == 0 {
			color.Yellow("No schemas found in %s", dir)
			return nil
		}
		fmt.Printf("Schemas detected in %s:\n", color.CyanString(dir))
		parts := strings.SplitN(pattern, "{type}", 2)
		for _, t := range types {
			filename := parts[0] + t + parts[1]
			fmt.Printf("  %s Type %s → %s\n", color.GreenString("✓"), t, filename)
		}
		return nil
	},
}

var schemaCheckCmd = &cobra.Command{
	Use:   "check <type>",
	Short: "Check that a schema is valid and loadable",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("schemas")
		pattern, _ := cmd.Flags().GetString("schema-pattern")
		if pattern == "" {
			pattern = viper.GetString("schema_pattern")
		}
		typeNode := args[0]
		loader, err := schema.NewLocalLoader(dir, pattern)
		if err != nil {
			return fmt.Errorf("invalid schema_pattern: %w", err)
		}
		_, err = loader.Load(typeNode)
		if err != nil {
			color.Red("❌ Invalid schema for type %s: %v", typeNode, err)
			return fmt.Errorf("invalid schema: %w", err)
		}
		color.Green("✅ Schema for type %s: OK", typeNode)
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
}
