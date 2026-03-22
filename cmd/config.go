// cmd/config.go
package cmd

import (
	"errors"
	"fmt"
	"io/fs"
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

# Schema filename pattern. Use {type} as placeholder for the type name.
# Default: json-schema-Node_{type}.json
# schema_pattern: json-schema-Node_{type}.json

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
		if err := os.WriteFile(filename, []byte(template), 0o644); err != nil {
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
		fmt.Printf("  schemas        : %s\n", viper.GetString("schemas"))
		fmt.Printf("  schema-pattern : %s\n", viper.GetString("schema_pattern"))
		fmt.Printf("  types          : %v\n", viper.GetStringSlice("types"))
		fmt.Printf("  output         : %s\n", viper.GetString("output"))
		fmt.Printf("  verbose        : %v\n", viper.GetBool("verbose"))
		fmt.Printf("  workers        : %d\n", viper.GetInt("workers"))
		fmt.Printf("  no-progress    : %v\n", viper.GetBool("no-progress"))
	},
}

// resolveConfigPath returns the target config file path.
func resolveConfigPath(global bool) (string, error) {
	if global {
		return globalConfigPath()
	}
	return ".nodeval.yaml", nil
}

// runConfigSet is the testable core of configSetCmd.
func runConfigSet(path, key, value string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	coerced, err := coerceValue(key, value)
	if err != nil {
		return err
	}
	m, err := readConfigFile(path)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}
	m[key] = coerced
	return writeConfigFile(path, m)
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration key in the local or global config file.

Valid keys: schemas, schema_pattern, output, verbose, workers, no_progress

Examples:
  nodeval config set schemas ./schemas
  nodeval config set output json
  nodeval config set --global workers 8`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		global, _ := cmd.Flags().GetBool("global")
		path, err := resolveConfigPath(global)
		if err != nil {
			return err
		}
		if err := runConfigSet(path, args[0], args[1]); err != nil {
			return err
		}
		coerced, _ := coerceValue(args[0], args[1])
		color.Green("✅ %s = %v (in %s)", args[0], coerced, path)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get the effective value of a configuration key",
	Long: `Print the effective value of a config key (CLI flags > local config > global config > defaults).

Valid keys: schemas, schema_pattern, output, verbose, workers, no_progress
Note: the "types" key is a list and is not supported by get/set/unset.

Examples:
  nodeval config get schemas
  nodeval config get output`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		if err := validateKey(key); err != nil {
			return err
		}
		switch validKeys[key] {
		case "bool":
			fmt.Fprintln(cmd.OutOrStdout(), viper.GetBool(key))
		case "int":
			fmt.Fprintln(cmd.OutOrStdout(), viper.GetInt(key))
		default:
			fmt.Fprintln(cmd.OutOrStdout(), viper.GetString(key))
		}
		return nil
	},
}

// runConfigUnset is the testable core of configUnsetCmd.
func runConfigUnset(path, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	// Explicitly check file existence: readConfigFile masks ErrNotExist and returns
	// an empty map, which would silently create the file on write. For unset, we
	// want an error when the file doesn't exist (unlike set, which creates it).
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("config file not found: %s", path)
	}
	m, err := readConfigFile(path)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}
	if _, ok := m[key]; !ok {
		color.Yellow("⚠️  key %q not set in %s", key, path)
		return nil
	}
	delete(m, key)
	return writeConfigFile(path, m)
}

var configUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Remove a configuration key",
	Long: `Remove a key from the local or global config file.
If the key is not present, a warning is printed and the command exits successfully.

Valid keys: schemas, schema_pattern, output, verbose, workers, no_progress
Note: the "types" key is a list and is not supported by get/set/unset.

Examples:
  nodeval config unset verbose
  nodeval config unset --global output`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		global, _ := cmd.Flags().GetBool("global")
		path, err := resolveConfigPath(global)
		if err != nil {
			return err
		}
		return runConfigUnset(path, args[0])
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configUnsetCmd)
	configSetCmd.Flags().Bool("global", false, "Write to global config (~/.config/nodeval/.nodeval.yaml)")
	configUnsetCmd.Flags().Bool("global", false, "Write to global config (~/.config/nodeval/.nodeval.yaml)")
}
