package reporter

import (
	"strings"
	"testing"

	"nodeval/internal/validator"
)

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
