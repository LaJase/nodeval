// internal/reporter/terminal.go
package reporter

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

var separator = strings.Repeat("-", 100)

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
	var totalFiles, totalErrors int
	for _, res := range r.Results {
		totalFiles += res.Success + res.Errors
		totalErrors += res.Errors
		prefix := color.GreenString(">")
		errStr := fmt.Sprintf("%d Errors", res.Errors)
		if res.Errors > 0 {
			prefix = color.RedString(">")
			errStr = color.RedString(errStr)
		}
		fmt.Printf("%s Nodes %-2s : %s | %s\n",
			prefix, res.Type,
			color.GreenString("%4d OK", res.Success),
			errStr,
		)
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
