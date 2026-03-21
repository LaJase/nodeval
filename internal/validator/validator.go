// internal/validator/validator.go
package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"jsnsch/internal/schema"
)

type FileError struct {
	File    string `json:"file"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

type TypeResult struct {
	Type    string      `json:"type"`
	Success int         `json:"success"`
	Errors  int         `json:"errors"`
	Details []FileError `json:"details"`
}

type result struct {
	mu sync.Mutex
	TypeResult
}

func (r *result) record(ok bool, fe FileError) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ok {
		r.Success++
	} else {
		r.Errors++
		r.Details = append(r.Details, fe)
	}
}

type task struct {
	path     string
	typeNode string
}

type Options struct {
	Workers    int
	Verbose    bool
	OnProgress func(typeNode string)
}

// Run validates all files in filesByType using the provided schema loader.
func Run(filesByType map[string][]string, loader schema.Loader, opts Options) []TypeResult {
	numWorkers := opts.Workers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	totalTasks := 0
	for _, files := range filesByType {
		totalTasks += len(files)
	}

	taskChan := make(chan task, totalTasks)
	resultsMap := make(map[string]*result)

	for typeNode, files := range filesByType {
		resultsMap[typeNode] = &result{TypeResult: TypeResult{Type: typeNode}}
		for _, p := range files {
			taskChan <- task{path: p, typeNode: typeNode}
		}
	}
	close(taskChan)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			schemaCache := make(map[string]*jsonschema.Schema)
			schemaFailed := make(map[string]bool)

			for t := range taskChan {
				if schemaFailed[t.typeNode] {
					if opts.OnProgress != nil {
						opts.OnProgress(t.typeNode)
					}
					continue
				}

				sch, exists := schemaCache[t.typeNode]
				if !exists {
					var err error
					sch, err = loader.Load(t.typeNode)
					if err != nil {
						schemaFailed[t.typeNode] = true
						resultsMap[t.typeNode].record(false, FileError{
							File:    fmt.Sprintf("json-schema-Node_%s.json", t.typeNode),
							Path:    "",
							Message: fmt.Sprintf("missing schema: %s", t.typeNode),
						})
						if opts.OnProgress != nil {
							opts.OnProgress(t.typeNode)
						}
						continue
					}
					schemaCache[t.typeNode] = sch
				}

				fe, ok := validateFile(sch, t.path, opts.Verbose)
				resultsMap[t.typeNode].record(ok, fe)
				if opts.OnProgress != nil {
					opts.OnProgress(t.typeNode)
				}
			}
		}()
	}
	wg.Wait()

	out := make([]TypeResult, 0, len(resultsMap))
	for _, r := range resultsMap {
		out = append(out, r.TypeResult)
	}
	return out
}

func validateFile(sch *jsonschema.Schema, fPath string, verbose bool) (FileError, bool) {
	baseName := filepath.Base(fPath)

	f, err := os.Open(fPath)
	if err != nil {
		return FileError{File: baseName, Message: "read error"}, false
	}
	defer f.Close()

	var v any
	if err := json.NewDecoder(f).Decode(&v); err != nil {
		return FileError{File: baseName, Message: "invalid JSON"}, false
	}

	errVal := sch.Validate(v)
	if errVal == nil {
		return FileError{}, true
	}

	ve, ok := errVal.(*jsonschema.ValidationError)
	if !ok {
		return FileError{File: baseName, Message: fmt.Sprintf("%v", errVal)}, false
	}

	errPath, msg := extractError(ve, verbose)
	return FileError{File: baseName, Path: errPath, Message: msg}, false
}

func extractError(ve *jsonschema.ValidationError, verbose bool) (path, msg string) {
	var contexts []string
	curr := ve
	for len(curr.Causes) > 0 {
		m := curr.Message
		if m != "" && !strings.Contains(m, "file://") {
			clean := strings.TrimPrefix(m, "doesn't validate with ")
			clean = strings.ReplaceAll(clean, "'", "")
			clean = strings.TrimPrefix(clean, "/definitions/")
			contexts = append(contexts, clean)
		}
		curr = curr.Causes[0]
	}

	finalMsg := curr.Message
	if strings.HasPrefix(finalMsg, "missing properties: ") {
		props := strings.TrimPrefix(finalMsg, "missing properties: ")
		finalMsg = fmt.Sprintf("%s are required", strings.ReplaceAll(props, "'", ""))
	}

	path = "root"
	if len(contexts) > 0 {
		path = strings.Join(contexts, " > ")
	}
	return path, finalMsg
}
