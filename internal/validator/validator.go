// Package validator runs concurrent JSON Schema validation over sets of files.
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
	"nodeval/internal/schema"
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

func (r *result) merge(b *localBatch) {
	r.mu.Lock()
	r.Success += b.success
	r.Errors += b.errors
	r.Details = append(r.Details, b.details...)
	r.mu.Unlock()
}

type localBatch struct {
	success int
	errors  int
	details []FileError
}

func (b *localBatch) add(ok bool, fe FileError) {
	if ok {
		b.success++
	} else {
		b.errors++
		b.details = append(b.details, fe)
	}
}

type task struct {
	path     string
	typeNode string
}

const progressBatchSize = 50

type Options struct {
	Workers    int
	Verbose    bool
	OnProgress func(typeNode string, count int)
}

// Run validates all files in filesByType using the provided schema loader.
func Run(filesByType map[string][]string, loader schema.Loader, opts Options) []TypeResult {
	numWorkers := opts.Workers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	taskChan := make(chan task, numWorkers*2)
	resultsMap := make(map[string]*result)

	for typeNode := range filesByType {
		resultsMap[typeNode] = &result{TypeResult: TypeResult{Type: typeNode}}
	}

	go func() {
		for typeNode, files := range filesByType {
			for _, p := range files {
				taskChan <- task{path: p, typeNode: typeNode}
			}
		}
		close(taskChan)
	}()

	var wg sync.WaitGroup
	for range numWorkers {
		wg.Go(func() {
			schemaCache := make(map[string]*jsonschema.Schema)
			schemaFailed := make(map[string]bool)
			batches := make(map[string]*localBatch, len(filesByType))
			for typeNode := range filesByType {
				batches[typeNode] = &localBatch{}
			}
			pendingProgress := make(map[string]int, len(filesByType))

			flush := func(typeNode string) {
				if opts.OnProgress != nil && pendingProgress[typeNode] > 0 {
					opts.OnProgress(typeNode, pendingProgress[typeNode])
					pendingProgress[typeNode] = 0
				}
			}

			for t := range taskChan {
				if schemaFailed[t.typeNode] {
					pendingProgress[t.typeNode]++
					if pendingProgress[t.typeNode] >= progressBatchSize {
						flush(t.typeNode)
					}
					continue
				}

				sch, exists := schemaCache[t.typeNode]
				if !exists {
					var err error
					sch, err = loader.Load(t.typeNode)
					if err != nil {
						schemaFailed[t.typeNode] = true
						batches[t.typeNode].add(false, FileError{
							File:    fmt.Sprintf("json-schema-Node_%s.json", t.typeNode),
							Path:    "",
							Message: fmt.Sprintf("missing schema: %s", t.typeNode),
						})
						pendingProgress[t.typeNode]++
						if pendingProgress[t.typeNode] >= progressBatchSize {
							flush(t.typeNode)
						}
						continue
					}
					schemaCache[t.typeNode] = sch
				}

				fe, ok := validateFile(sch, t.path)
				batches[t.typeNode].add(ok, fe)
				pendingProgress[t.typeNode]++
				if pendingProgress[t.typeNode] >= progressBatchSize {
					flush(t.typeNode)
				}
			}

			for typeNode := range pendingProgress {
				flush(typeNode)
			}
			for typeNode, b := range batches {
				resultsMap[typeNode].merge(b)
			}
		})
	}
	wg.Wait()

	out := make([]TypeResult, 0, len(resultsMap))
	for _, r := range resultsMap {
		out = append(out, r.TypeResult)
	}
	return out
}

func validateFile(sch *jsonschema.Schema, fPath string) (FileError, bool) {
	baseName := filepath.Base(fPath)

	data, err := os.ReadFile(fPath)
	if err != nil {
		return FileError{File: baseName, Message: "read error"}, false
	}

	var v any
	if err := json.Unmarshal(data, &v); err != nil {
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

	errPath, msg := extractError(ve)
	return FileError{File: baseName, Path: errPath, Message: msg}, false
}

func extractError(ve *jsonschema.ValidationError) (path, msg string) {
	contexts := make([]string, 0, 4)
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
	if props, ok := strings.CutPrefix(finalMsg, "missing properties: "); ok {
		finalMsg = fmt.Sprintf("%s are required", strings.ReplaceAll(props, "'", ""))
	}

	path = "root"
	if len(contexts) > 0 {
		path = strings.Join(contexts, " > ")
	}
	return path, finalMsg
}
