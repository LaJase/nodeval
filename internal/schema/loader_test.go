package schema_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"nodeval/internal/schema"
)

func TestLocalLoaderMissing(t *testing.T) {
	loader := schema.NewLocalLoader(t.TempDir())
	_, err := loader.Load("X")
	if err == nil {
		t.Error("expected error for missing schema")
	}
}

func TestLocalLoaderValid(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`{"type": "object"}`)
	_ = os.WriteFile(filepath.Join(dir, "json-schema-Node_M.json"), content, 0644)

	loader := schema.NewLocalLoader(dir)
	sch, err := loader.Load("M")
	if err != nil {
		t.Fatal(err)
	}
	if sch == nil {
		t.Error("expected non-nil schema")
	}
}

func TestLocalLoader_CachesSchema(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`{"type": "object"}`)
	_ = os.WriteFile(filepath.Join(dir, "json-schema-Node_M.json"), content, 0644)

	loader := schema.NewLocalLoader(dir)

	const n = 16
	addrs := make([]string, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sch, err := loader.Load("M")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			addrs[idx] = fmt.Sprintf("%p", sch)
		}(i)
	}
	wg.Wait()

	for i := 1; i < n; i++ {
		if addrs[i] != addrs[0] {
			t.Error("Load returned different schema instances: schema is not cached globally")
		}
	}
}
