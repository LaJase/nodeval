// cmd/validate.go
package cmd

import (
	"fmt"
	"runtime"
	"sort"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"

	"nodeval/internal/reporter"
	"nodeval/internal/scanner"
	"nodeval/internal/schema"
	"nodeval/internal/validator"
)

var validateCmd = &cobra.Command{
	Use:   "validate <directory>",
	Short: "Validate JSON files in a directory against their schemas",
	Long: `Recursively walks <directory> and validates each *_<TYPE>.json file
against the corresponding json-schema-Node_<TYPE>.json schema.

Examples:
  nodeval validate ./data --all
  nodeval validate ./data --types M,R --verbose
  nodeval validate ./data --all --output json > results.json
  nodeval validate ./data --all --output junit > results.xml`,
	Args: cobra.ExactArgs(1),
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)

	f := validateCmd.Flags()
	f.String("schemas", ".", "Directory containing JSON schemas")
	f.StringSlice("types", nil, "Types to validate (e.g. M,R,I). Default: auto-detected")
	f.Bool("all", false, "Validate all detected types")
	f.String("output", "terminal", "Output format: terminal | json | junit")
	f.Bool("verbose", false, "Show full validation error details")
	f.Int("workers", 0, "Number of workers (0 = NumCPU)")
	f.Bool("no-progress", false, "Disable progress bars")
	f.String("schema-pattern", "", "Schema filename pattern (e.g. schema_{type}.json)")

	_ = viper.BindPFlags(f)
}

func runValidate(cmd *cobra.Command, args []string) error {
	dir := args[0]
	schemasDir := viper.GetString("schemas")
	typesFlag := viper.GetStringSlice("types")
	allFlag := viper.GetBool("all")
	outputFmt := viper.GetString("output")
	verbose := viper.GetBool("verbose")
	workers := viper.GetInt("workers")
	noProgress := viper.GetBool("no-progress")
	schemaPattern := viper.GetString("schema-pattern")
	if schemaPattern == "" {
		schemaPattern = viper.GetString("schema_pattern")
	}

	// Resolve types
	var types []string
	if allFlag || len(typesFlag) == 0 {
		detected, err := schema.DetectTypes(schemasDir, schemaPattern)
		if err != nil {
			return fmt.Errorf("type detection: %w", err)
		}
		if len(typesFlag) > 0 && !allFlag {
			types = typesFlag
		} else {
			types = detected
		}
	} else {
		types = typesFlag
	}

	if len(types) == 0 {
		return fmt.Errorf("no types found in %s — check --schemas or use --types", schemasDir)
	}

	if outputFmt == "terminal" {
		fmt.Printf("\n🚀 Analyzing : %s\n", color.CyanString(dir))
		fmt.Printf("📂 Schemas   : %s\n", color.CyanString(schemasDir))
		fmt.Printf("🏷️  Types     : %v\n\n", types)
	}

	// Scan files
	scanStart := time.Now()
	filesByType, err := scanner.ScanFiles(dir, types)
	if err != nil {
		return fmt.Errorf("directory scan: %w", err)
	}
	scanDuration := time.Since(scanStart)
	if outputFmt == "terminal" {
		fmt.Printf("🔎 Scan time   : %v\n", scanDuration.Round(time.Millisecond))
	}

	totalTasks := 0
	for _, files := range filesByType {
		totalTasks += len(files)
	}
	if totalTasks == 0 {
		color.Yellow("⚠️  No files found for the requested types.")
		return nil
	}

	// Sort types for consistent display
	typeOrder := make(map[string]int, len(types))
	for i, t := range types {
		typeOrder[t] = i
	}

	// Setup progress bars
	var p *mpb.Progress
	bars := make(map[string]*mpb.Bar)
	if outputFmt == "terminal" && !noProgress {
		p = mpb.New(mpb.WithWidth(60))
		for _, t := range types {
			files := filesByType[t]
			if len(files) == 0 {
				continue
			}
			name := fmt.Sprintf("🔍 [Type %s]", t)
			bars[t] = p.AddBar(int64(len(files)),
				mpb.PrependDecorators(
					decor.Name(name, decor.WC{W: runewidth.StringWidth(name) + 1}),
					decor.CountersNoUnit("%d/%d", decor.WC{W: 10}),
				),
				mpb.AppendDecorators(
					decor.Percentage(decor.WC{W: 5}),
					decor.OnComplete(decor.Name(""), color.GreenString("  ✅")),
				),
			)
		}
	}

	numWorkers := workers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	if outputFmt == "terminal" {
		fmt.Printf("👷 Active workers : %d\n", numWorkers)
	}

	start := time.Now()
	loader, err := schema.NewLocalLoader(schemasDir, schemaPattern)
	if err != nil {
		return &ConfigError{Msg: fmt.Sprintf("invalid schema_pattern: %v", err)}
	}
	results := validator.Run(filesByType, loader, validator.Options{
		Workers: numWorkers,
		Verbose: verbose,
		OnProgress: func(typeNode string, count int) {
			if b, ok := bars[typeNode]; ok {
				b.IncrBy(count)
			}
		},
	})

	if p != nil {
		p.Wait()
	}
	duration := time.Since(start)

	// Sort results
	sort.Slice(results, func(i, j int) bool {
		return typeOrder[results[i].Type] < typeOrder[results[j].Type]
	})

	// Render
	report := reporter.Report{Duration: duration, Results: results}
	var r reporter.Reporter
	switch outputFmt {
	case "json":
		r = &reporter.JSON{}
	case "junit":
		r = &reporter.JUnit{}
	default:
		r = &reporter.Terminal{Verbose: verbose}
	}

	if err := r.Render(report); err != nil {
		return err
	}

	// Exit code
	for _, res := range results {
		if res.Errors > 0 {
			return &ValidationError{Msg: fmt.Sprintf("%d invalid file(s)", res.Errors)}
		}
	}
	return nil
}
