package schema

import (
	"fmt"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Loader interface {
	Load(typeNode string) (*jsonschema.Schema, error)
}

type LocalLoader struct {
	dir string
}

func NewLocalLoader(dir string) *LocalLoader {
	return &LocalLoader{dir: dir}
}

func (l *LocalLoader) Load(typeNode string) (*jsonschema.Schema, error) {
	path := filepath.Join(l.dir, fmt.Sprintf("json-schema-Node_%s.json", typeNode))
	compiler := jsonschema.NewCompiler()
	sch, err := compiler.Compile(path)
	if err != nil {
		return nil, fmt.Errorf("schema %s: %w", typeNode, err)
	}
	return sch, nil
}
