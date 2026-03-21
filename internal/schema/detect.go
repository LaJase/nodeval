package schema

import (
	"os"
	"sort"
	"strings"
)

const schemaPrefix = "json-schema-Node_"

// DetectTypes scans dir and returns all type names found from schema filenames.
func DetectTypes(dir string) ([]string, error) {
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
		if strings.HasPrefix(name, schemaPrefix) && strings.HasSuffix(name, ".json") {
			t := strings.TrimPrefix(name, schemaPrefix)
			t = strings.TrimSuffix(t, ".json")
			if t != "" {
				types = append(types, t)
			}
		}
	}
	sort.Strings(types)
	return types, nil
}
