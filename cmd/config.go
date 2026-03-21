// cmd/config.go
package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Gérer la configuration de jsnsch",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Générer un fichier .jsnsch.yaml exemple dans le dossier courant",
	RunE: func(cmd *cobra.Command, args []string) error {
		const template = `# jsnsch configuration
# Documentation: jsnsch --help

# Dossier contenant les schémas JSON (json-schema-Node_<TYPE>.json)
schemas: .

# Types à valider. Si vide, auto-détecté depuis le dossier schemas.
# types:
#   - M
#   - R
#   - I

# Format de sortie: terminal | json | junit
output: terminal

# Afficher le détail complet des erreurs de validation
verbose: false

# Nombre de workers parallèles (0 = NumCPU automatique)
workers: 0

# Désactiver les barres de progression (utile en CI/CD)
no_progress: false
`
		const filename = ".jsnsch.yaml"
		if _, err := os.Stat(filename); err == nil {
			color.Yellow("⚠️  %s existe déjà. Supprimez-le avant de relancer init.", filename)
			return nil
		}
		if err := os.WriteFile(filename, []byte(template), 0644); err != nil {
			return fmt.Errorf("impossible de créer %s: %w", filename, err)
		}
		color.Green("✅ %s créé avec succès.", filename)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Afficher la configuration active (fusion CLI + fichier + défauts)",
	Run: func(cmd *cobra.Command, args []string) {
		cfgUsed := viper.ConfigFileUsed()
		if cfgUsed == "" {
			cfgUsed = "(aucun fichier de config trouvé)"
		}
		fmt.Printf("Fichier de config : %s\n\n", color.CyanString(cfgUsed))
		fmt.Printf("  schemas     : %s\n", viper.GetString("schemas"))
		fmt.Printf("  types       : %v\n", viper.GetStringSlice("types"))
		fmt.Printf("  output      : %s\n", viper.GetString("output"))
		fmt.Printf("  verbose     : %v\n", viper.GetBool("verbose"))
		fmt.Printf("  workers     : %d\n", viper.GetInt("workers"))
		fmt.Printf("  no-progress : %v\n", viper.GetBool("no-progress"))
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
}
