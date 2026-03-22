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
	loader, err := schema.NewLocalLoader(t.TempDir(), "json-schema-Node_{type}.json")
	if err != nil {
		t.Fatal(err)
	}
	_, err = loader.Load("X")
	if err == nil {
		t.Error("expected error for missing schema")
	}
}

func TestLocalLoaderValid(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`{"type": "object"}`)
	_ = os.WriteFile(filepath.Join(dir, "json-schema-Node_M.json"), content, 0644)

	loader, err := schema.NewLocalLoader(dir, "json-schema-Node_{type}.json")
	if err != nil {
		t.Fatal(err)
	}
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

	loader, err := schema.NewLocalLoader(dir, "json-schema-Node_{type}.json")
	if err != nil {
		t.Fatal(err)
	}

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

func TestLocalLoader_CustomPattern(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "schema_M.json"), []byte(`{"type": "object"}`), 0644)

	loader, err := schema.NewLocalLoader(dir, "schema_{type}.json")
	if err != nil {
		t.Fatal(err)
	}
	sch, err := loader.Load("M")
	if err != nil || sch == nil {
		t.Fatalf("expected valid schema, got err=%v", err)
	}
}

func TestLocalLoader_InvalidPattern(t *testing.T) {
	_, err := schema.NewLocalLoader(t.TempDir(), "schema.json")
	if err == nil {
		t.Error("expected error for pattern without {type}")
	}
}
