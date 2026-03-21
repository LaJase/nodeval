// cmd/schema.go
package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"jsnsch/internal/schema"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Gérer et inspecter les schémas JSON",
}

var schemaListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lister les schémas détectés dans le dossier --schemas",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("schemas")
		types, err := schema.DetectTypes(dir)
		if err != nil {
			return fmt.Errorf("lecture du dossier schémas: %w", err)
		}
		if len(types) == 0 {
			color.Yellow("Aucun schéma trouvé dans %s", dir)
			return nil
		}
		fmt.Printf("Schémas détectés dans %s:\n", color.CyanString(dir))
		for _, t := range types {
			fmt.Printf("  %s Type %s → json-schema-Node_%s.json\n", color.GreenString("✓"), t, t)
		}
		return nil
	},
}

var schemaCheckCmd = &cobra.Command{
	Use:   "check <type>",
	Short: "Vérifier qu'un schéma est valide et chargeable",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := cmd.Flags().GetString("schemas")
		typeNode := args[0]
		loader := schema.NewLocalLoader(dir)
		_, err := loader.Load(typeNode)
		if err != nil {
			color.Red("❌ Schéma invalide pour le type %s: %v", typeNode, err)
			return fmt.Errorf("schéma invalide: %w", err)
		}
		color.Green("✅ Schéma pour le type %s : OK", typeNode)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(schemaCmd)
	schemaCmd.AddCommand(schemaListCmd)
	schemaCmd.AddCommand(schemaCheckCmd)

	schemaListCmd.Flags().String("schemas", ".", "Dossier contenant les schémas")
	schemaCheckCmd.Flags().String("schemas", ".", "Dossier contenant les schémas")
}
