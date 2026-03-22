package reporter

import (
	"encoding/json"
	"io"
	"os"
)

type jsonOutput struct {
	DurationMs int64            `json:"duration_ms"`
	Total      int              `json:"total"`
	Errors     int              `json:"errors"`
	Success    bool             `json:"success"`
	Results    []jsonTypeResult `json:"results"`
}

type jsonTypeResult struct {
	Type    string       `json:"type"`
	Success int          `json:"success"`
	Errors  int          `json:"errors"`
	Details []jsonDetail `json:"details,omitempty"`
}

type jsonDetail struct {
	File    string `json:"file"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

type JSON struct {
	Writer io.Writer
}

func (j *JSON) Render(r Report) error {
	w := j.Writer
	if w == nil {
		w = os.Stdout
	}

	var total, errs int
	results := make([]jsonTypeResult, 0, len(r.Results))
	for _, res := range r.Results {
		total += res.Success + res.Errors
		errs += res.Errors
		tr := jsonTypeResult{
			Type:    res.Type,
			Success: res.Success,
			Errors:  res.Errors,
		}
		for _, d := range res.Details {
			tr.Details = append(tr.Details, jsonDetail{File: d.File, Path: d.Path, Message: d.Message})
		}
		results = append(results, tr)
	}

	out := jsonOutput{
		DurationMs: r.Duration.Milliseconds(),
		Total:      total,
		Errors:     errs,
		Success:    errs == 0,
		Results:    results,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
