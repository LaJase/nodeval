// internal/reporter/reporter.go
package reporter

import (
	"jsnsch/internal/validator"
	"time"
)

type Report struct {
	Duration time.Duration
	Results  []validator.TypeResult
}

type Reporter interface {
	Render(r Report) error
}
