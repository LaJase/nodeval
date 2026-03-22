// cmd/schema.go
package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
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
		types, err := schema.DetectTypes(dir, "json-schema-Node_{type}.json")
		if err != nil {
			return fmt.Errorf("reading schemas directory: %w", err)
		}
		if len(types) == 0 {
			color.Yellow("No schemas found in %s", dir)
			return nil
		}
		fmt.Printf("Schemas detected in %s:\n", color.CyanString(dir))
		for _, t := range types {
			fmt.Printf("  %s Type %s → json-schema-Node_%s.json\n", color.GreenString("✓"), t, t)
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
		typeNode := args[0]
		loader := schema.NewLocalLoader(dir)
		_, err := loader.Load(typeNode)
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
}
