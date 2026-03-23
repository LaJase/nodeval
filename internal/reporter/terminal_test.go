package reporter

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"

	"nodeval/internal/validator"
)

func captureStdout(f func()) string {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	f()
	w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestSummaryLines_Aligned(t *testing.T) {
	results := []validator.TypeResult{
		{Type: "Alpha", Success: 16, Errors: 4},
		{Type: "Zeta", Success: 16, Errors: 4},
	}
	width := maxTypeWidth(results)
	lines := make([]string, len(results))
	for i, res := range results {
		lines[i] = summaryLine(width, res)
	}

	pos0 := strings.Index(lines[0], "|")
	pos1 := strings.Index(lines[1], "|")
	if pos0 < 0 || pos0 != pos1 {
		t.Errorf("summary lines not aligned: '|' at %d vs %d\n  %q\n  %q",
			pos0, pos1, lines[0], lines[1])
	}
}

func TestTerminal_NormalMode_SingleError_ShowsPathAndMessage(t *testing.T) {
	report := Report{
		Duration: time.Second,
		Results: []validator.TypeResult{
			{Type: "T", Success: 0, Errors: 1, Details: []validator.FileError{
				{File: "foo_T.json", Path: "id", Message: "expected integer"},
			}},
		},
	}
	out := captureStdout(func() {
		tr := &Terminal{Verbose: false}
		_ = tr.Render(report)
	})
	if !strings.Contains(out, "foo_T.json") {
		t.Error("expected file name in output")
	}
	if !strings.Contains(out, "id") {
		t.Error("expected path in output")
	}
	if !strings.Contains(out, "expected integer") {
		t.Error("expected message in output")
	}
}

func TestTerminal_NormalMode_MultipleErrors_ShowsCount(t *testing.T) {
	report := Report{
		Duration: time.Second,
		Results: []validator.TypeResult{
			{Type: "T", Success: 0, Errors: 1, Details: []validator.FileError{
				{File: "bar_T.json", Count: 3},
			}},
		},
	}
	out := captureStdout(func() {
		tr := &Terminal{Verbose: false}
		_ = tr.Render(report)
	})
	if !strings.Contains(out, "bar_T.json") {
		t.Error("expected file name in output")
	}
	if !strings.Contains(out, "3 errors") {
		t.Errorf("expected '3 errors' in output, got:\n%s", out)
	}
}

func TestTerminal_NormalMode_CountBoundary_TwoErrors(t *testing.T) {
	report := Report{
		Duration: time.Second,
		Results: []validator.TypeResult{
			{Type: "T", Success: 0, Errors: 1, Details: []validator.FileError{
				{File: "x_T.json", Count: 2},
			}},
		},
	}
	out := captureStdout(func() {
		tr := &Terminal{Verbose: false}
		_ = tr.Render(report)
	})
	if !strings.Contains(out, "2 errors") {
		t.Errorf("expected '2 errors' for Count=2, got:\n%s", out)
	}
}

func TestTerminal_VerboseMode_GroupsErrorsByFile(t *testing.T) {
	report := Report{
		Duration: time.Second,
		Results: []validator.TypeResult{
			{Type: "T", Success: 0, Errors: 1, Details: []validator.FileError{
				{File: "baz_T.json", Path: "id", Message: "expected integer"},
				{File: "baz_T.json", Path: "name", Message: "expected string"},
			}},
		},
	}
	out := captureStdout(func() {
		tr := &Terminal{Verbose: true}
		_ = tr.Render(report)
	})
	// File name should appear exactly once in verbose output (grouped header)
	fileCount := strings.Count(out, "baz_T.json")
	if fileCount != 1 {
		t.Errorf("expected file name once in verbose output, got %d occurrences:\n%s", fileCount, out)
	}
	if !strings.Contains(out, "id") || !strings.Contains(out, "name") {
		t.Error("expected both error paths in verbose output")
	}
}
