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

	"nodeval/internal/schema"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type FileError struct {
	File    string `json:"file"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message,omitempty"`
	Count   int    `json:"count,omitempty"` // >1 when multiple errors exist but details were not extracted (normal mode)
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

// add records one validated file. fes must be non-empty when ok is false.
func (b *localBatch) add(ok bool, fes []FileError) {
	if ok {
		b.success++
	} else {
		b.errors++
		b.details = append(b.details, fes...)
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
						batches[t.typeNode].add(false, []FileError{{
							File:    t.typeNode,
							Message: fmt.Sprintf("missing schema: %s", t.typeNode),
						}})
						pendingProgress[t.typeNode]++
						if pendingProgress[t.typeNode] >= progressBatchSize {
							flush(t.typeNode)
						}
						continue
					}
					schemaCache[t.typeNode] = sch
				}

				fes, ok := validateFile(sch, t.path, opts.Verbose)
				batches[t.typeNode].add(ok, fes)
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

func validateFile(sch *jsonschema.Schema, fPath string, verbose bool) ([]FileError, bool) {
	baseName := filepath.Base(fPath)

	data, err := os.ReadFile(fPath)
	if err != nil {
		return []FileError{{File: baseName, Message: fmt.Sprintf("read error: %v", err)}}, false
	}

	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return []FileError{{File: baseName, Message: fmt.Sprintf("invalid JSON: %v", err)}}, false
	}

	errVal := sch.Validate(v)
	if errVal == nil {
		return nil, true
	}

	ve, ok := errVal.(*jsonschema.ValidationError)
	if !ok {
		return []FileError{{File: baseName, Message: fmt.Sprintf("%v", errVal)}}, false
	}

	if verbose {
		fes := extractAllErrors(ve)
		for i := range fes {
			fes[i].File = baseName
		}
		return fes, false
	}

	// Normal mode: avoid allocating path/message strings when there are multiple errors.
	count := countLeafErrors(ve)
	if count == 1 {
		errPath, msg := extractError(ve)
		return []FileError{{File: baseName, Path: errPath, Message: msg}}, false
	}
	return []FileError{{File: baseName, Count: count}}, false
}

// formatMessage applies a human-friendly transformation to a raw schema error message.
func formatMessage(msg string) string {
	if props, ok := strings.CutPrefix(msg, "missing properties: "); ok {
		return fmt.Sprintf("%s are required", strings.ReplaceAll(props, "'", ""))
	}
	return msg
}

// countLeafErrors returns the number of leaf causes in a ValidationError tree.
// It performs no string allocations, making it safe to call in normal (non-verbose) mode.
func countLeafErrors(ve *jsonschema.ValidationError) int {
	if len(ve.Causes) == 0 {
		return 1
	}
	n := 0
	for _, c := range ve.Causes {
		n += countLeafErrors(c)
	}
	return n
}

// extractAllErrors traverses all leaf causes and returns one FileError per leaf.
// Intermediate nodes that have both a message and sub-causes are intentionally skipped —
// the actionable error detail lives at the leaves for the property/type violations this
// tool validates against.
// File is left empty; the caller sets it.
func extractAllErrors(ve *jsonschema.ValidationError) []FileError {
	if len(ve.Causes) == 0 {
		path := jsonPtrToDot(ve.InstanceLocation)
		msg := formatMessage(ve.Message)
		return []FileError{{Path: path, Message: msg}}
	}
	var result []FileError
	for _, c := range ve.Causes {
		result = append(result, extractAllErrors(c)...)
	}
	return result
}

func extractError(ve *jsonschema.ValidationError) (path, msg string) {
	curr := ve
	for len(curr.Causes) > 0 {
		curr = curr.Causes[0]
	}
	return jsonPtrToDot(curr.InstanceLocation), formatMessage(curr.Message)
}

// jsonPtrToDot converts a JSON Pointer (RFC 6901) to dot/bracket notation.
// Example: /users/0/age → users[0].age
func jsonPtrToDot(ptr string) string {
	if ptr == "" {
		return "(root)"
	}
	parts := strings.Split(strings.TrimPrefix(ptr, "/"), "/")
	var b strings.Builder
	for i, p := range parts {
		p = strings.ReplaceAll(p, "~1", "/")
		p = strings.ReplaceAll(p, "~0", "~")
		switch {
		case i == 0:
			b.WriteString(p)
		case isAllDigits(p):
			b.WriteString("[")
			b.WriteString(p)
			b.WriteString("]")
		default:
			b.WriteString(".")
			b.WriteString(p)
		}
	}
	return b.String()
}

func isAllDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
