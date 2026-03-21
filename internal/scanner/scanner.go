package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

// ScanFiles walks dir recursively and returns files matching *_<type>.json per type.
func ScanFiles(dir string, types []string) (map[string][]string, error) {
	filesByType := make(map[string][]string)
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		for _, t := range types {
			if strings.HasSuffix(d.Name(), "_"+t+".json") {
				filesByType[t] = append(filesByType[t], path)
			}
		}
		return nil
	})
	return filesByType, err
}
