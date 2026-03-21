// internal/reporter/junit_test.go
package reporter_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"nodeval/internal/reporter"
	"nodeval/internal/validator"
)

func TestJUnitReporter(t *testing.T) {
	var buf bytes.Buffer
	r := &reporter.JUnit{Writer: &buf}

	report := reporter.Report{
		Duration: 200 * time.Millisecond,
		Results: []validator.TypeResult{
			{Type: "M", Success: 5, Errors: 1, Details: []validator.FileError{
				{File: "bad_M.json", Path: "root > address", Message: "street is required"},
			}},
		},
	}
	if err := r.Render(report); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "<testsuites") {
		t.Error("expected <testsuites> element")
	}
	if !strings.Contains(out, "bad_M.json") {
		t.Error("expected failing test case for bad_M.json")
	}
	if !strings.Contains(out, "street is required") {
		t.Error("expected failure message")
	}
}
