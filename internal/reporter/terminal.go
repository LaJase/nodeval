package reporter

import (
	"fmt"
	"strings"

	"github.com/fatih/color"

	"nodeval/internal/validator"
)

const verboseIndent = "   "

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

type group struct {
	file   string
	errors []validator.FileError
}

func (t *Terminal) Render(r Report) error {
	fmt.Printf("\n%s\n", separator)

	for _, res := range r.Results {
		if t.Verbose {
			// Group FileErrors by file (preserve order of first appearance).
			seen := make(map[string]int)
			var groups []group
			for _, d := range res.Details {
				if i, ok := seen[d.File]; ok {
					groups[i].errors = append(groups[i].errors, d)
				} else {
					seen[d.File] = len(groups)
					groups = append(groups, group{file: d.File, errors: []validator.FileError{d}})
				}
			}
			for _, g := range groups {
				fmt.Printf("%s %s :\n", color.RedString("❌"), color.YellowString(g.file))
				for _, e := range g.errors {
					fmt.Printf("%s%s : %s\n", verboseIndent, e.Path, e.Message)
				}
				fmt.Println()
			}
		} else {
			for _, d := range res.Details {
				if d.Count > 1 {
					fmt.Printf("%s %s : %d errors\n",
						color.RedString("❌"),
						color.YellowString(d.File),
						d.Count,
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
