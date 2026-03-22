// Package reporter provides output formatters for validation results.
package reporter

import (
	"time"

	"nodeval/internal/validator"
)

type Report struct {
	Duration time.Duration
	Results  []validator.TypeResult
}

type Reporter interface {
	Render(r Report) error
}
