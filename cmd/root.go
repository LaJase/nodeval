// cmd/root.go
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"nodeval/internal/config"
)

var cfgFile string

// ValidationError indicates that at least one file is invalid (exit 1)
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

// ConfigError indicates a config/missing schema problem (exit 2)
type ConfigError struct{ Msg string }

func (e *ConfigError) Error() string { return e.Msg }

var rootCmd = &cobra.Command{
	Use:          "nodeval",
	SilenceUsage: true,
	Short: "Multithreaded JSON Schema validator",
	Long: `nodeval validates JSON files against their associated schemas.

Files must follow the naming convention *_<TYPE>.json.
Schemas must be named json-schema-Node_<TYPE>.json.

Examples:
  nodeval validate ./data --all
  nodeval validate ./data --types M,R,I --output json
  nodeval schema list --schemas ./schemas
  nodeval config init`,
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: .nodeval.yaml)")
}

func initConfig() {
	// Inject defaults from config.Default()
	defaults := config.Default()
	viper.SetDefault("schemas", defaults.Schemas)
	viper.SetDefault("output", defaults.Output)
	viper.SetDefault("workers", defaults.Workers)
	viper.SetDefault("verbose", defaults.Verbose)
	viper.SetDefault("no_progress", defaults.NoProgress)
	viper.SetDefault("schema_pattern", defaults.SchemaPattern)
	viper.SetDefault("directory", "")

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, _ := os.UserHomeDir()
		viper.AddConfigPath(".")
		viper.AddConfigPath(filepath.Join(home, ".config", "nodeval"))
		viper.SetConfigName(".nodeval")
		viper.SetConfigType("yaml")
	}
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()
}
