// cmd/root.go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

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
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "fichier de config (défaut: .jsnsch.yaml)")
}

func initConfig() {
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
