// Package reporter provides output formatters for validation results.
package reporter

import (
	"io"
	"os"
	"time"

	"nodeval/internal/validator"
)

const (
	FormatTerminal = "terminal"
	FormatJSON     = "json"
	FormatJUnit    = "junit"
)

type Report struct {
	Duration time.Duration
	Results  []validator.TypeResult
}

type Reporter interface {
	Render(r Report) error
}

// effectiveWriter returns w if non-nil, otherwise os.Stdout.
func effectiveWriter(w io.Writer) io.Writer {
	if w == nil {
		return os.Stdout
	}
	return w
}

// calculateTotals returns the total file count and total error count across all results.
func calculateTotals(results []validator.TypeResult) (totalFiles, totalErrors int) {
	for _, res := range results {
		totalFiles += res.Success + res.Errors
		totalErrors += res.Errors
	}
	return
}
