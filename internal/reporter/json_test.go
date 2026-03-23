// internal/reporter/json_test.go
package reporter_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"nodeval/internal/reporter"
	"nodeval/internal/validator"
)

func TestJSONReporter(t *testing.T) {
	var buf bytes.Buffer
	r := &reporter.JSON{Writer: &buf}

	report := reporter.Report{
		Duration: 500 * time.Millisecond,
		Results: []validator.TypeResult{
			{Type: "M", Success: 10, Errors: 1, Details: []validator.FileError{
				{File: "a_M.json", Path: "root", Message: "x is required"},
			}},
		},
	}
	if err := r.Render(report); err != nil {
		t.Fatal(err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if out["total"].(float64) != 11 {
		t.Errorf("expected total=11, got %v", out["total"])
	}
	if out["errors"].(float64) != 1 {
		t.Errorf("expected errors=1, got %v", out["errors"])
	}
	if out["success"].(bool) != false {
		t.Error("expected success=false")
	}
}

func TestJSON_CountOnlyError(t *testing.T) {
	var buf bytes.Buffer
	r := &reporter.JSON{Writer: &buf}
	err := r.Render(reporter.Report{
		Results: []validator.TypeResult{
			{Type: "T", Errors: 1, Details: []validator.FileError{
				{File: "x_T.json", Count: 3},
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, `"count": 3`) {
		t.Errorf("expected count in JSON output, got:\n%s", out)
	}
	if strings.Contains(out, `"path"`) || strings.Contains(out, `"message"`) {
		t.Errorf("expected no path/message in count-only output, got:\n%s", out)
	}
}
