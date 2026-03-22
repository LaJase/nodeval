package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"go.yaml.in/yaml/v3"
)

// validKeys maps settable config keys to their type ("string", "bool", "int").
// The "types" key is intentionally excluded: it is a list and requires a
// different input format not supported by the set/unset commands.
var validKeys = map[string]string{
	"directory":      "string",
	"schemas":        "string",
	"schema_pattern": "string",
	"output":         "string",
	"verbose":        "bool",
	"workers":        "int",
	"no_progress":    "bool",
}

func validateKey(key string) error {
	if _, ok := validKeys[key]; !ok {
		return fmt.Errorf("unknown config key: %q", key)
	}
	return nil
}

func coerceValue(key, raw string) (any, error) {
	typ, ok := validKeys[key]
	if !ok {
		return nil, fmt.Errorf("unknown config key: %q", key)
	}
	switch typ {
	case "bool":
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("key %q expects a bool (true/false), got %q", key, raw)
		}
		return v, nil
	case "int":
		v, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("key %q expects an integer, got %q", key, raw)
		}
		return v, nil
	default:
		return raw, nil
	}
}

// globalConfigPath returns the path to the global config file.
func globalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "nodeval", ".nodeval.yaml"), nil
}

// readConfigFile reads a YAML config file into a map.
// Returns an empty map (no error) if the file does not exist.
func readConfigFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

// writeConfigFile marshals m to YAML and writes it to path,
// creating parent directories as needed.
func writeConfigFile(path string, m map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
