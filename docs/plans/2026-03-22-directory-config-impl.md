# Directory Config Option — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the `<directory>` argument of `nodeval validate` optional by allowing it to be set in config via `directory` key.

**Architecture:** Four small changes: add `mapstructure` tag to `Config.Directory`, add `"directory"` to `validKeys`, wire Viper default in `root.go`, update `configShowCmd`/`configInitCmd` template, and make the positional arg optional in `validate.go` with a clear error if neither is provided.

**Tech Stack:** Go 1.25, Cobra, Viper

---

### Task 1 : `internal/config/config.go` — tag mapstructure

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Step 1 : Écrire le test qui échoue**

Dans `internal/config/config_test.go`, ajouter :

```go
func TestLoadFromFile_Directory(t *testing.T) {
	dir := t.TempDir()
	content := []byte("directory: ./data\n")
	_ = os.WriteFile(filepath.Join(dir, ".nodeval.yaml"), content, 0o644)

	cfg, err := config.LoadFrom(filepath.Join(dir, ".nodeval.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Directory != "./data" {
		t.Errorf("expected directory=./data, got %q", cfg.Directory)
	}
}
```

**Step 2 : Vérifier que le test échoue**

```bash
go test ./internal/config/... -run TestLoadFromFile_Directory -v
```

Attendu : FAIL — `cfg.Directory` vaut `""` car le tag `mapstructure` est absent.

**Step 3 : Implémenter**

Dans `internal/config/config.go`, modifier le champ `Directory` :

```go
type Config struct {
	Directory     string   `mapstructure:"directory"`
	Schemas       string   `mapstructure:"schemas"`
	SchemaPattern string   `mapstructure:"schema_pattern"`
	Types         []string `mapstructure:"types"`
	All           bool     `mapstructure:"all"`
	Output        string   `mapstructure:"output"`
	Verbose       bool     `mapstructure:"verbose"`
	Workers       int      `mapstructure:"workers"`
	NoProgress    bool     `mapstructure:"no_progress"`
}
```

**Step 4 : Vérifier que les tests passent**

```bash
go test ./internal/config/... -v
```

Attendu : tous PASS

**Step 5 : Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add mapstructure tag to Config.Directory"
```

---

### Task 2 : `cmd/config_rw.go` + `cmd/root.go` — wiring

**Files:**
- Modify: `cmd/config_rw.go`
- Modify: `cmd/root.go`
- Test: `cmd/config_rw_test.go`

**Step 1 : Écrire le test qui échoue**

Dans `cmd/config_rw_test.go`, ajouter :

```go
func TestValidateKey_Directory(t *testing.T) {
	if err := validateKey("directory"); err != nil {
		t.Errorf("expected directory to be a valid key, got: %v", err)
	}
}
```

**Step 2 : Vérifier que le test échoue**

```bash
go test ./cmd/... -run TestValidateKey_Directory -v
```

Attendu : FAIL — `validateKey("directory")` retourne une erreur.

**Step 3 : Implémenter**

Dans `cmd/config_rw.go`, ajouter `"directory"` à `validKeys` :

```go
var validKeys = map[string]string{
	"directory":      "string",
	"schemas":        "string",
	"schema_pattern": "string",
	"output":         "string",
	"verbose":        "bool",
	"workers":        "int",
	"no_progress":    "bool",
}
```

Dans `cmd/root.go`, dans `initConfig()`, après les autres `viper.SetDefault` :

```go
viper.SetDefault("directory", "")
```

**Step 4 : Vérifier que les tests passent**

```bash
go test ./cmd/... -run TestValidateKey_Directory -v
go build ./...
```

Attendu : PASS

**Step 5 : Commit**

```bash
git add cmd/config_rw.go cmd/root.go cmd/config_rw_test.go
git commit -m "feat: add directory to validKeys and viper defaults"
```

---

### Task 3 : `cmd/config.go` — show + init template

**Files:**
- Modify: `cmd/config.go`

**Step 1 : Mettre à jour `configShowCmd`**

Dans `configShowCmd.Run`, ajouter la ligne `directory` **avant** `schemas` (premier dans la liste) :

```go
fmt.Printf("  directory      : %s\n", viper.GetString("directory"))
fmt.Printf("  schemas        : %s\n", viper.GetString("schemas"))
```

**Step 2 : Mettre à jour le template de `configInitCmd`**

Dans la const `template` de `configInitCmd`, ajouter après le commentaire d'en-tête :

```
# Directory containing the JSON files to validate.
# When set, nodeval validate can be run without a positional argument.
# directory: ./data
```

**Step 3 : Mettre à jour le `Long` de `configSetCmd`**

Remplacer :
```
Valid keys: schemas, schema_pattern, output, verbose, workers, no_progress
```
Par :
```
Valid keys: directory, schemas, schema_pattern, output, verbose, workers, no_progress
```

Faire la même chose dans le `Long` de `configGetCmd` et `configUnsetCmd`.

**Step 4 : Vérifier le build**

```bash
go build ./...
go test ./cmd/... -v
```

Attendu : PASS

**Step 5 : Commit**

```bash
git add cmd/config.go
git commit -m "feat: add directory to config show, init template, and command help"
```

---

### Task 4 : `cmd/validate.go` — arg optionnel

**Files:**
- Modify: `cmd/validate.go`

**Step 1 : Écrire les tests qui échouent**

Créer `cmd/validate_dir_test.go` :

```go
package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func execValidateDir(args []string, configDir string) error {
	viper.Reset()
	viper.SetDefault("directory", configDir)
	viper.SetDefault("schemas", ".")
	viper.SetDefault("output", "terminal")
	viper.SetDefault("workers", 0)
	viper.SetDefault("verbose", false)
	viper.SetDefault("no_progress", false)
	viper.SetDefault("schema_pattern", "json-schema-Node_{type}.json")

	root := &cobra.Command{Use: "nodeval"}
	child := &cobra.Command{
		Use:  "validate [directory]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := viper.GetString("directory")
			if len(args) > 0 {
				dir = args[0]
			}
			if dir == "" {
				return fmt.Errorf("no directory specified — pass it as argument or set 'directory' in config")
			}
			return nil
		},
	}
	root.AddCommand(child)
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(append([]string{"validate"}, args...))
	return root.Execute()
}

func TestValidate_ArgOverridesConfig(t *testing.T) {
	if err := execValidateDir([]string{"./data"}, "./other"); err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
}

func TestValidate_ConfigUsedWhenNoArg(t *testing.T) {
	if err := execValidateDir([]string{}, "./data"); err != nil {
		t.Fatalf("expected success with config dir, got: %v", err)
	}
}

func TestValidate_ErrorWhenNeitherArgNorConfig(t *testing.T) {
	err := execValidateDir([]string{}, "")
	if err == nil {
		t.Fatal("expected error when no dir and no config")
	}
	if !strings.Contains(err.Error(), "no directory specified") {
		t.Errorf("unexpected error: %v", err)
	}
}
```

Ajouter les imports manquants : `"fmt"` et `"strings"`.

**Step 2 : Vérifier que les tests échouent**

```bash
go test ./cmd/... -run "TestValidate_" -v
```

Attendu : erreur de compilation (`execValidateDir` non défini)

**Step 3 : Implémenter dans `cmd/validate.go`**

Modifier `validateCmd` :

```go
var validateCmd = &cobra.Command{
	Use:   "validate [directory]",
	Short: "Validate JSON files in a directory against their schemas",
	Long: `Recursively walks <directory> and validates each *_<TYPE>.json file
against the corresponding json-schema-Node_<TYPE>.json schema.

The directory can be set in config (nodeval config set directory ./data)
and omitted from the command line.

Examples:
  nodeval validate ./data --all
  nodeval validate ./data --types M,R --verbose
  nodeval validate --all
  nodeval validate ./data --all --output json > results.json
  nodeval validate ./data --all --output junit > results.xml`,
	Args: cobra.MaximumNArgs(1),
	RunE: runValidate,
}
```

Dans `runValidate`, remplacer :

```go
dir := args[0]
```

par :

```go
dir := viper.GetString("directory")
if len(args) > 0 {
    dir = args[0]
}
if dir == "" {
    return fmt.Errorf("no directory specified — pass it as argument or set 'directory' in config")
}
```

**Step 4 : Vérifier que les tests passent**

```bash
go test ./cmd/... -run "TestValidate_" -v
go test ./...
go build ./...
```

Attendu : tous PASS

**Step 5 : Commit**

```bash
git add cmd/validate.go cmd/validate_dir_test.go
git commit -m "feat: make validate directory optional via config"
```

---

### Task 5 : README

**Files:**
- Modify: `README.md`

**Step 1 : Mettre à jour la section `nodeval validate`**

Changer `Use: "validate <directory>"` → `"validate [directory]"` dans l'exemple de commande.

Ajouter un exemple sans argument :

```bash
# Directory set in config, run without argument
nodeval validate --all
```

**Step 2 : Mettre à jour la table de config**

Dans la table des clés config, ajouter en premier :

```markdown
| `directory` | string | _(empty)_ | Default directory for JSON files to validate. |
```

**Step 3 : Mettre à jour la section `.nodeval.yaml` example**

Ajouter après le commentaire d'en-tête :

```yaml
# Directory containing the JSON files to validate.
# When set, nodeval validate can be run without a positional argument.
# directory: ./data
```

**Step 4 : Vérifier le build final**

```bash
go build ./... && go test ./...
```

**Step 5 : Commit**

```bash
git add README.md
git commit -m "docs: document directory config option"
```
