// Package scanner walks directories and groups JSON files by node type.
package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

// ScanFiles walks dir recursively and returns files matching *_<type>.json per type.
func ScanFiles(dir string, types []string) (map[string][]string, error) {
	typeSet := make(map[string]struct{}, len(types))
	for _, t := range types {
		typeSet[t] = struct{}{}
	}

	filesByType := make(map[string][]string)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() || !strings.HasSuffix(name, ".json") {
			return nil
		}
		base := name[:len(name)-5] // strip ".json"
		idx := strings.LastIndex(base, "_")
		if idx < 0 {
			return nil
		}
		t := base[idx+1:]
		if _, ok := typeSet[t]; ok {
			filesByType[t] = append(filesByType[t], path)
		}
		return nil
	})
	return filesByType, err
}
