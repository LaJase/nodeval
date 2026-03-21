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
	Short: "Manage nodeval configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a sample .nodeval.yaml file in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		const template = `# nodeval configuration
# Documentation: nodeval --help

# Directory containing JSON schemas (json-schema-Node_<TYPE>.json)
schemas: .

# Types to validate. If empty, auto-detected from the schemas directory.
# types:
#   - M
#   - R
#   - I

# Output format: terminal | json | junit
output: terminal

# Show full validation error details
verbose: false

# Number of parallel workers (0 = NumCPU automatic)
workers: 0

# Disable progress bars (useful in CI/CD)
no_progress: false
`
		const filename = ".nodeval.yaml"
		if _, err := os.Stat(filename); err == nil {
			color.Yellow("⚠️  %s already exists. Remove it before running init again.", filename)
			return nil
		}
		if err := os.WriteFile(filename, []byte(template), 0644); err != nil {
			return fmt.Errorf("unable to create %s: %w", filename, err)
		}
		color.Green("✅ %s created successfully.", filename)
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the active configuration (merge of CLI + file + defaults)",
	Run: func(cmd *cobra.Command, args []string) {
		cfgUsed := viper.ConfigFileUsed()
		if cfgUsed == "" {
			cfgUsed = "(no config file found)"
		}
		fmt.Printf("Config file : %s\n\n", color.CyanString(cfgUsed))
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
