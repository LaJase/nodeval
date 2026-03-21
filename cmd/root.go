// cmd/root.go
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"jsnsch/internal/config"
)

var cfgFile string

// ValidationError indique qu'au moins un fichier est invalide (exit 1)
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

// ConfigError indique un problème de config/schéma manquant (exit 2)
type ConfigError struct{ Msg string }

func (e *ConfigError) Error() string { return e.Msg }

var rootCmd = &cobra.Command{
	Use:   "jsnsch",
	Short: "Validateur JSON Schema multithreadé",
	Long: `jsnsch valide des fichiers JSON contre leurs schémas associés.

Les fichiers doivent suivre la convention *_<TYPE>.json.
Les schémas doivent être nommés json-schema-Node_<TYPE>.json.

Exemples:
  jsnsch validate ./data --all
  jsnsch validate ./data --types M,R,I --output json
  jsnsch schema list --schemas ./schemas
  jsnsch config init`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		var ve *ValidationError
		var ce *ConfigError
		switch {
		case errors.As(err, &ve):
			os.Exit(1)
		case errors.As(err, &ce):
			os.Exit(2)
		default:
			fmt.Fprintln(os.Stderr, err)
			os.Exit(3)
		}
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "fichier de config (défaut: .jsnsch.yaml)")
}

func initConfig() {
	// Injecter les defaults depuis config.Default()
	defaults := config.Default()
	viper.SetDefault("schemas", defaults.Schemas)
	viper.SetDefault("output", defaults.Output)
	viper.SetDefault("workers", defaults.Workers)
	viper.SetDefault("verbose", defaults.Verbose)
	viper.SetDefault("no_progress", defaults.NoProgress)

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, _ := os.UserHomeDir()
		viper.AddConfigPath(".")
		viper.AddConfigPath(filepath.Join(home, ".config", "jsnsch"))
		viper.SetConfigName(".jsnsch")
		viper.SetConfigType("yaml")
	}
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()
}
