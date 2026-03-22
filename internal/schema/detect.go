// Package schema handles JSON Schema loading and type detection.
package schema

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// ParsePattern splits pattern on "{type}" and returns prefix and suffix.
// Returns an error if "{type}" is not present.
func ParsePattern(pattern string) (prefix, suffix string, err error) {
	parts := strings.SplitN(pattern, "{type}", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("schema_pattern %q must contain {type}", pattern)
	}
	return parts[0], parts[1], nil
}

// DetectTypes scans dir and returns all type names matching the given pattern.
func DetectTypes(dir, pattern string) ([]string, error) {
	prefix, suffix, err := ParsePattern(pattern)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var types []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			t := name[len(prefix) : len(name)-len(suffix)]
			if t != "" {
				types = append(types, t)
			}
		}
	}
	sort.Strings(types)
	return types, nil
}
