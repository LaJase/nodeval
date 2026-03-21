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

	"jsnsch/internal/reporter"
	"jsnsch/internal/scanner"
	"jsnsch/internal/schema"
	"jsnsch/internal/validator"
)

var validateCmd = &cobra.Command{
	Use:   "validate <directory>",
	Short: "Valide les fichiers JSON d'un dossier contre leurs schémas",
	Long: `Parcourt <directory> récursivement et valide chaque fichier *_<TYPE>.json
contre le schéma json-schema-Node_<TYPE>.json correspondant.

Exemples:
  jsnsch validate ./data --all
  jsnsch validate ./data --types M,R --verbose
  jsnsch validate ./data --all --output json > results.json
  jsnsch validate ./data --all --output junit > results.xml`,
	Args: cobra.ExactArgs(1),
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)

	f := validateCmd.Flags()
	f.String("schemas", ".", "Dossier contenant les schémas JSON")
	f.StringSlice("types", nil, "Types à valider (ex: M,R,I). Défaut: auto-détecté")
	f.Bool("all", false, "Valider tous les types détectés")
	f.String("output", "terminal", "Format de sortie: terminal | json | junit")
	f.Bool("verbose", false, "Afficher le détail complet des erreurs")
	f.Int("workers", 0, "Nombre de workers (0 = NumCPU)")
	f.Bool("no-progress", false, "Désactiver les barres de progression")

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

	// Resolve types
	var types []string
	if allFlag || len(typesFlag) == 0 {
		detected, err := schema.DetectTypes(schemasDir)
		if err != nil {
			return fmt.Errorf("détection des types: %w", err)
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
		return fmt.Errorf("aucun type trouvé dans %s — vérifiez --schemas ou utilisez --types", schemasDir)
	}

	if outputFmt == "terminal" {
		fmt.Printf("\n🚀 Analyse de : %s\n", color.CyanString(dir))
		fmt.Printf("📂 Schémas   : %s\n", color.CyanString(schemasDir))
		fmt.Printf("🏷️  Types     : %v\n\n", types)
	}

	// Scan files
	filesByType, err := scanner.ScanFiles(dir, types)
	if err != nil {
		return fmt.Errorf("scan du dossier: %w", err)
	}

	totalTasks := 0
	for _, files := range filesByType {
		totalTasks += len(files)
	}
	if totalTasks == 0 {
		color.Yellow("⚠️  Aucun fichier trouvé pour les types demandés.")
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
		fmt.Printf("👷 Workers actifs : %d\n", numWorkers)
	}

	start := time.Now()
	results := validator.Run(filesByType, schema.NewLocalLoader(schemasDir), validator.Options{
		Workers: numWorkers,
		Verbose: verbose,
		OnProgress: func(typeNode string) {
			if b, ok := bars[typeNode]; ok {
				b.Increment()
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
			return &ValidationError{Msg: fmt.Sprintf("%d fichier(s) invalide(s)", res.Errors)}
		}
	}
	return nil
}
