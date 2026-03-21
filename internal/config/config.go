package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Directory  string
	Schemas    string   `mapstructure:"schemas"`
	Types      []string `mapstructure:"types"`
	All        bool     `mapstructure:"all"`
	Output     string   `mapstructure:"output"`
	Verbose    bool     `mapstructure:"verbose"`
	Workers    int      `mapstructure:"workers"`
	NoProgress bool     `mapstructure:"no_progress"`
}

func Default() Config {
	return Config{
		Schemas:    ".",
		Output:     "terminal",
		Workers:    0,
		Verbose:    false,
		NoProgress: false,
	}
}

func LoadFrom(path string) (Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return Default(), err
	}
	cfg := Default()
	if err := v.Unmarshal(&cfg); err != nil {
		return Default(), err
	}
	return cfg, nil
}
