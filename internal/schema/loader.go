// Package schema handles JSON Schema loading and type detection.
package schema

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Loader interface {
	Load(typeNode string) (*jsonschema.Schema, error)
}

type LocalLoader struct {
	dir      string
	prefix   string
	suffix   string
	mu       sync.RWMutex
	cache    map[string]*jsonschema.Schema
	compiler *jsonschema.Compiler
}

func NewLocalLoader(dir, pattern string) (*LocalLoader, error) {
	prefix, suffix, err := parsePattern(pattern)
	if err != nil {
		return nil, err
	}
	return &LocalLoader{
		dir:      dir,
		prefix:   prefix,
		suffix:   suffix,
		cache:    make(map[string]*jsonschema.Schema),
		compiler: jsonschema.NewCompiler(),
	}, nil
}

func (l *LocalLoader) Load(typeNode string) (*jsonschema.Schema, error) {
	l.mu.RLock()
	sch, found := l.cache[typeNode]
	l.mu.RUnlock()
	if found {
		return sch, nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	// Double-check after acquiring write lock.
	if sch, found = l.cache[typeNode]; found {
		return sch, nil
	}

	path := filepath.Join(l.dir, l.prefix+typeNode+l.suffix)
	compiled, err := l.compiler.Compile(path)
	if err != nil {
		return nil, fmt.Errorf("schema %s: %w", typeNode, err)
	}
	l.cache[typeNode] = compiled
	return compiled, nil
}
