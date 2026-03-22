package reporter

import (
	"fmt"
	"strings"

	"github.com/fatih/color"

	"nodeval/internal/validator"
)

var separator = strings.Repeat("-", 100)

func maxTypeWidth(results []validator.TypeResult) int {
	w := 0
	for _, res := range results {
		if len(res.Type) > w {
			w = len(res.Type)
		}
	}
	return w
}

func summaryLine(typeWidth int, res validator.TypeResult) string {
	prefix := color.GreenString(">")
	errStr := fmt.Sprintf("%d Errors", res.Errors)
	if res.Errors > 0 {
		prefix = color.RedString(">")
		errStr = color.RedString(errStr)
	}
	return fmt.Sprintf("%s Nodes %-*s : %s | %s",
		prefix, typeWidth, res.Type,
		color.GreenString("%4d OK", res.Success),
		errStr,
	)
}

type Terminal struct {
	Verbose bool
}

func (t *Terminal) Render(r Report) error {
	fmt.Printf("\n%s\n", separator)

	for _, res := range r.Results {
		for _, d := range res.Details {
			if t.Verbose {
				fmt.Printf("%s %s\n    Path    : %s\n    Message : %s\n\n",
					color.RedString("❌"),
					color.YellowString(d.File),
					d.Path,
					d.Message,
				)
			} else {
				fmt.Printf("%s %s : %s : %s\n",
					color.RedString("❌"),
					color.YellowString(d.File),
					d.Path,
					d.Message,
				)
			}
		}
	}

	fmt.Printf("\nSummary:\n")
	totalFiles, totalErrors := calculateTotals(r.Results)
	w := maxTypeWidth(r.Results)
	for _, res := range r.Results {
		fmt.Println(summaryLine(w, res))
	}

	fmt.Printf("\n%s\n\n", separator)
	finalMsg := fmt.Sprintf("TOTAL : %d files analyzed | %d errors", totalFiles, totalErrors)
	fmt.Printf("⏱️  Total time : %v\n", r.Duration.Round(1000000))

	if totalErrors == 0 {
		color.Green("⭐ " + finalMsg + " (VALID)")
	} else {
		color.Red("🚨 " + finalMsg + " (INVALID)")
	}
	return nil
}
