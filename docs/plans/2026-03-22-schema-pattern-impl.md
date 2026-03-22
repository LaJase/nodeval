# Schema Pattern Configurable — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rendre le pattern de nommage des fichiers schema configurable via `--schema-pattern` / `schema_pattern` dans `.nodeval.yaml`, avec `json-schema-Node_{type}.json` comme défaut.

**Architecture:** Le placeholder `{type}` est splitté en prefix/suffix au moment de l'initialisation (`NewLocalLoader`, `DetectTypes`). La validation du pattern (présence de `{type}`) se fait à ce moment-là, pas au runtime. La config `SchemaPattern` suit le même chemin Viper que les autres clés.

**Tech Stack:** Go 1.25, cobra, viper, `strings.SplitN`

---

### Task 1 : Ajouter `SchemaPattern` à la config

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Step 1 : Écrire le test qui échoue**

Dans `internal/config/config_test.go`, ajouter :

```go
func TestDefaultSchemaPattern(t *testing.T) {
    cfg := config.Default()
    if cfg.SchemaPattern != "json-schema-Node_{type}.json" {
        t.Errorf("expected default schema pattern, got %q", cfg.SchemaPattern)
    }
}
```

**Step 2 : Vérifier que le test échoue**

```bash
go test ./internal/config/... -run TestDefaultSchemaPattern -v
```

Attendu : `FAIL — cfg.SchemaPattern undefined`

**Step 3 : Implémenter**

Dans `internal/config/config.go`, ajouter le champ et la valeur par défaut :

```go
type Config struct {
    Directory     string
    Schemas       string   `mapstructure:"schemas"`
    SchemaPattern string   `mapstructure:"schema_pattern"`
    Types         []string `mapstructure:"types"`
    All           bool     `mapstructure:"all"`
    Output        string   `mapstructure:"output"`
    Verbose       bool     `mapstructure:"verbose"`
    Workers       int      `mapstructure:"workers"`
    NoProgress    bool     `mapstructure:"no_progress"`
}

func Default() Config {
    return Config{
        Schemas:       ".",
        SchemaPattern: "json-schema-Node_{type}.json",
        Output:        "terminal",
        Workers:       0,
        Verbose:       false,
        NoProgress:    false,
    }
}
```

**Step 4 : Vérifier que le test passe**

```bash
go test ./internal/config/... -v
```

Attendu : tous les tests `PASS`

**Step 5 : Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add SchemaPattern to config with default json-schema-Node_{type}.json"
```

---

### Task 2 : Helper `parsePattern` + mise à jour de `DetectTypes`

**Files:**
- Modify: `internal/schema/detect.go`
- Test: `internal/schema/detect_test.go`

**Step 1 : Écrire les tests qui échouent**

Dans `internal/schema/detect_test.go`, ajouter :

```go
func TestDetectTypes_CustomPattern(t *testing.T) {
    dir := t.TempDir()
    for _, name := range []string{"schema_M.json", "schema_R.json", "unrelated.json"} {
        _ = os.WriteFile(filepath.Join(dir, name), []byte(`{}`), 0644)
    }

    types, err := schema.DetectTypes(dir, "schema_{type}.json")
    if err != nil {
        t.Fatal(err)
    }
    if len(types) != 2 || !slices.Contains(types, "M") || !slices.Contains(types, "R") {
        t.Errorf("expected [M R], got %v", types)
    }
}

func TestDetectTypes_InvalidPattern(t *testing.T) {
    _, err := schema.DetectTypes(t.TempDir(), "schema.json")
    if err == nil {
        t.Error("expected error for pattern without {type}")
    }
}
```

Mettre à jour le test existant `TestDetectTypes` pour passer le pattern par défaut :

```go
types, err := schema.DetectTypes(dir, "json-schema-Node_{type}.json")
```

**Step 2 : Vérifier que les nouveaux tests échouent**

```bash
go test ./internal/schema/... -run "TestDetectTypes" -v
```

Attendu : erreur de compilation (`DetectTypes` n'accepte pas 2 args)

**Step 3 : Implémenter**

Remplacer `internal/schema/detect.go` entièrement :

```go
// Package schema handles JSON Schema loading and type detection.
package schema

import (
    "fmt"
    "os"
    "sort"
    "strings"
)

// parsePattern splits pattern on "{type}" and returns prefix and suffix.
// Returns an error if "{type}" is not present.
func parsePattern(pattern string) (prefix, suffix string, err error) {
    parts := strings.SplitN(pattern, "{type}", 2)
    if len(parts) != 2 {
        return "", "", fmt.Errorf("schema_pattern %q must contain {type}", pattern)
    }
    return parts[0], parts[1], nil
}

// DetectTypes scans dir and returns all type names matching the given pattern.
func DetectTypes(dir, pattern string) ([]string, error) {
    prefix, suffix, err := parsePattern(pattern)
    if err != nil {
        return nil, err
    }

    entries, err := os.ReadDir(dir)
    if err != nil {
        return nil, err
    }

    var types []string
    for _, e := range entries {
        if e.IsDir() {
            continue
        }
        name := e.Name()
        if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
            t := name[len(prefix) : len(name)-len(suffix)]
            if t != "" {
                types = append(types, t)
            }
        }
    }
    sort.Strings(types)
    return types, nil
}
```

**Step 4 : Vérifier que tous les tests passent**

```bash
go test ./internal/schema/... -v
```

Attendu : tous les tests `PASS`

**Step 5 : Commit**

```bash
git add internal/schema/detect.go internal/schema/detect_test.go
git commit -m "feat: DetectTypes accepts configurable pattern with {type} placeholder"
```

---

### Task 3 : Mise à jour de `LocalLoader`

**Files:**
- Modify: `internal/schema/loader.go`
- Test: `internal/schema/loader_test.go`

**Step 1 : Écrire les tests qui échouent**

Dans `internal/schema/loader_test.go`, ajouter :

```go
func TestLocalLoader_CustomPattern(t *testing.T) {
    dir := t.TempDir()
    _ = os.WriteFile(filepath.Join(dir, "schema_M.json"), []byte(`{"type": "object"}`), 0644)

    loader, err := schema.NewLocalLoader(dir, "schema_{type}.json")
    if err != nil {
        t.Fatal(err)
    }
    sch, err := loader.Load("M")
    if err != nil || sch == nil {
        t.Fatalf("expected valid schema, got err=%v", err)
    }
}

func TestLocalLoader_InvalidPattern(t *testing.T) {
    _, err := schema.NewLocalLoader(t.TempDir(), "schema.json")
    if err == nil {
        t.Error("expected error for pattern without {type}")
    }
}
```

Mettre à jour les tests existants qui appellent `schema.NewLocalLoader(dir)` :

```go
// TestLocalLoaderMissing
loader, err := schema.NewLocalLoader(t.TempDir(), "json-schema-Node_{type}.json")
if err != nil { t.Fatal(err) }

// TestLocalLoaderValid
loader, err := schema.NewLocalLoader(dir, "json-schema-Node_{type}.json")
if err != nil { t.Fatal(err) }

// TestLocalLoader_CachesSchema
loader, err := schema.NewLocalLoader(dir, "json-schema-Node_{type}.json")
if err != nil { t.Fatal(err) }
```

**Step 2 : Vérifier que les nouveaux tests échouent**

```bash
go test ./internal/schema/... -run "TestLocalLoader" -v
```

Attendu : erreur de compilation (`NewLocalLoader` n'accepte pas 2 args)

**Step 3 : Implémenter**

Dans `internal/schema/loader.go`, mettre à jour `LocalLoader` et `NewLocalLoader` :

```go
type LocalLoader struct {
    dir      string
    prefix   string
    suffix   string
    mu       sync.RWMutex
    cache    map[string]*jsonschema.Schema
    compiler *jsonschema.Compiler
}

func NewLocalLoader(dir, pattern string) (*LocalLoader, error) {
    prefix, suffix, err := parsePattern(pattern)
    if err != nil {
        return nil, err
    }
    return &LocalLoader{
        dir:      dir,
        prefix:   prefix,
        suffix:   suffix,
        cache:    make(map[string]*jsonschema.Schema),
        compiler: jsonschema.NewCompiler(),
    }, nil
}
```

Dans `Load()`, remplacer la construction du path :

```go
// Remplacer :
path := filepath.Join(l.dir, fmt.Sprintf("json-schema-Node_%s.json", typeNode))
// Par :
path := filepath.Join(l.dir, l.prefix+typeNode+l.suffix)
```

Mettre aussi à jour le message d'erreur `missing schema` dans `validator.go` pour ne plus hardcoder le nom de fichier — utiliser seulement le typeNode :

```go
// Dans internal/validator/validator.go, remplacer :
File: fmt.Sprintf("json-schema-Node_%s.json", t.typeNode),
// Par :
File: t.typeNode,
```

**Step 4 : Vérifier que tous les tests passent**

```bash
go test ./internal/... -v
```

Attendu : tous les tests `PASS`

**Step 5 : Commit**

```bash
git add internal/schema/loader.go internal/schema/loader_test.go internal/validator/validator.go
git commit -m "feat: NewLocalLoader accepts configurable pattern with {type} placeholder"
```

---

### Task 4 : Mise à jour de `cmd/root.go`

**Files:**
- Modify: `cmd/root.go`

**Step 1 : Ajouter le default Viper**

Dans `initConfig()`, après les autres `viper.SetDefault`, ajouter :

```go
viper.SetDefault("schema_pattern", defaults.SchemaPattern)
```

**Step 2 : Vérifier le build**

```bash
go build ./...
```

**Step 3 : Commit**

```bash
git add cmd/root.go
git commit -m "feat: add schema_pattern viper default"
```

---

### Task 5 : Mise à jour de `cmd/validate.go`

**Files:**
- Modify: `cmd/validate.go`

**Step 1 : Ajouter le flag**

Dans `init()`, après les autres flags :

```go
f.String("schema-pattern", "", "Schema filename pattern (e.g. schema_{type}.json)")
```

**Step 2 : Lire et utiliser le pattern dans `runValidate`**

Après `workers := viper.GetInt("workers")`, ajouter :

```go
schemaPattern := viper.GetString("schema-pattern")
if schemaPattern == "" {
    schemaPattern = viper.GetString("schema_pattern")
}
```

Remplacer l'appel à `schema.DetectTypes` :

```go
// Remplacer :
detected, err := schema.DetectTypes(schemasDir)
// Par :
detected, err := schema.DetectTypes(schemasDir, schemaPattern)
```

Remplacer l'appel à `schema.NewLocalLoader` :

```go
// Remplacer :
results := validator.Run(filesByType, schema.NewLocalLoader(schemasDir), validator.Options{
// Par :
loader, err := schema.NewLocalLoader(schemasDir, schemaPattern)
if err != nil {
    return &ConfigError{Msg: fmt.Sprintf("invalid schema_pattern: %v", err)}
}
results := validator.Run(filesByType, loader, validator.Options{
```

**Step 3 : Vérifier le build et les tests**

```bash
go build ./... && go test ./...
```

**Step 4 : Commit**

```bash
git add cmd/validate.go
git commit -m "feat: add --schema-pattern flag to validate command"
```

---

### Task 6 : Mise à jour de `cmd/schema.go`

**Files:**
- Modify: `cmd/schema.go`

**Step 1 : Ajouter le flag persistent**

Dans `init()`, après `schemaCmd.PersistentFlags().String("schemas", ...)` :

```go
schemaCmd.PersistentFlags().String("schema-pattern", "", "Schema filename pattern (e.g. schema_{type}.json)")
```

**Step 2 : Mettre à jour `schemaListCmd`**

```go
RunE: func(cmd *cobra.Command, args []string) error {
    dir, _ := cmd.Flags().GetString("schemas")
    pattern, _ := cmd.Flags().GetString("schema-pattern")
    if pattern == "" {
        pattern = viper.GetString("schema_pattern")
    }

    types, err := schema.DetectTypes(dir, pattern)
    if err != nil {
        return fmt.Errorf("reading schemas directory: %w", err)
    }
    if len(types) == 0 {
        color.Yellow("No schemas found in %s", dir)
        return nil
    }
    fmt.Printf("Schemas detected in %s:\n", color.CyanString(dir))
    // Construire le nom de fichier depuis le pattern
    parts := strings.SplitN(pattern, "{type}", 2)
    for _, t := range types {
        filename := parts[0] + t + parts[1]
        fmt.Printf("  %s Type %s → %s\n", color.GreenString("✓"), t, filename)
    }
    return nil
},
```

Ajouter `"strings"` aux imports si absent.

**Step 3 : Mettre à jour `schemaCheckCmd`**

```go
RunE: func(cmd *cobra.Command, args []string) error {
    dir, _ := cmd.Flags().GetString("schemas")
    pattern, _ := cmd.Flags().GetString("schema-pattern")
    if pattern == "" {
        pattern = viper.GetString("schema_pattern")
    }
    typeNode := args[0]
    loader, err := schema.NewLocalLoader(dir, pattern)
    if err != nil {
        return fmt.Errorf("invalid schema_pattern: %w", err)
    }
    _, err = loader.Load(typeNode)
    if err != nil {
        color.Red("❌ Invalid schema for type %s: %v", typeNode, err)
        return fmt.Errorf("invalid schema: %w", err)
    }
    color.Green("✅ Schema for type %s: OK", typeNode)
    return nil
},
```

**Step 4 : Vérifier le build et les tests**

```bash
go build ./... && go test ./...
```

**Step 5 : Commit**

```bash
git add cmd/schema.go
git commit -m "feat: schema list/check use configurable pattern"
```

---

### Task 7 : Mise à jour de `cmd/config.go`

**Files:**
- Modify: `cmd/config.go`

**Step 1 : Ajouter `schema_pattern` dans `configShowCmd`**

Dans `configShowCmd`, après la ligne `schemas` :

```go
fmt.Printf("  schema-pattern : %s\n", viper.GetString("schema_pattern"))
```

**Step 2 : Ajouter `schema_pattern` dans le template `configInitCmd`**

Dans la const `template`, après la section `schemas` :

```
# Schema filename pattern. Use {type} as placeholder for the type name.
# Default: json-schema-Node_{type}.json
# schema_pattern: json-schema-Node_{type}.json
```

**Step 3 : Vérifier le build**

```bash
go build ./... && go test ./...
```

**Step 4 : Commit**

```bash
git add cmd/config.go
git commit -m "feat: show schema_pattern in config show and init template"
```

---

### Task 8 : Mettre à jour le README

**Files:**
- Modify: `README.md`

**Step 1 : Ajouter `--schema-pattern` dans la table des flags de `validate`**

```markdown
| `--schema-pattern` | _(défaut config)_ | Schema filename pattern. Use `{type}` as placeholder. |
```

**Step 2 : Mettre à jour la section "Schema naming convention"**

Ajouter une note que le pattern est configurable :

```markdown
The schema pattern defaults to `json-schema-Node_{type}.json` and can be
overridden with `--schema-pattern` or `schema_pattern` in `.nodeval.yaml`.
```

**Step 3 : Mettre à jour la table de config**

Ajouter une ligne :

```markdown
| `schema_pattern` | string | `json-schema-Node_{type}.json` | Schema filename pattern (`{type}` = type name). |
```

**Step 4 : Commit**

```bash
git add README.md
git commit -m "docs: document schema_pattern config option"
```
