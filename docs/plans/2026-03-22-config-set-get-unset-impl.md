# Config set/get/unset — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `nodeval config set`, `nodeval config get`, and `nodeval config unset` following the `git config` model.

**Architecture:** New helpers in `cmd/config_rw.go` handle YAML read/write; three new sub-commands in `cmd/config.go` use them. `--global` flag on `set` and `unset` selects the target file. `get` reads the merged Viper value.

**Tech Stack:** Go 1.25, Cobra, Viper, `go.yaml.in/yaml/v3`

---

### Task 1 : Helpers `cmd/config_rw.go`

**Files:**
- Create: `cmd/config_rw.go`
- Test: `cmd/config_rw_test.go`

**Step 1 : Écrire les tests qui échouent**

Créer `cmd/config_rw_test.go` :

```go
package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadWriteConfigFile_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")

	data := map[string]any{"schemas": "./schemas", "workers": 4}
	if err := writeConfigFile(path, data); err != nil {
		t.Fatal(err)
	}

	got, err := readConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got["schemas"] != "./schemas" {
		t.Errorf("expected schemas=./schemas, got %v", got["schemas"])
	}
}

func TestReadConfigFile_Missing(t *testing.T) {
	got, err := readConfigFile("/nonexistent/path/.nodeval.yaml")
	if err != nil {
		t.Fatal("expected empty map, not error")
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestValidateKey(t *testing.T) {
	if err := validateKey("schemas"); err != nil {
		t.Errorf("expected schemas to be valid: %v", err)
	}
	if err := validateKey("unknown_key"); err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestCoerceValue(t *testing.T) {
	v, err := coerceValue("workers", "4")
	if err != nil || v != 4 {
		t.Errorf("expected int 4, got %v err %v", v, err)
	}
	v, err = coerceValue("verbose", "true")
	if err != nil || v != true {
		t.Errorf("expected bool true, got %v err %v", v, err)
	}
	v, err = coerceValue("schemas", "./data")
	if err != nil || v != "./data" {
		t.Errorf("expected string ./data, got %v err %v", v, err)
	}
}
```

**Step 2 : Vérifier que les tests échouent**

```bash
go test ./cmd/... -run "TestReadWrite|TestValidate|TestCoerce" -v
```

Attendu : erreur de compilation (`readConfigFile`, `writeConfigFile`, `validateKey`, `coerceValue` non définis)

**Step 3 : Implémenter**

Créer `cmd/config_rw.go` :

```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"go.yaml.in/yaml/v3"
)

// validKeys maps each config key to its type: "string", "bool", or "int".
var validKeys = map[string]string{
	"schemas":        "string",
	"schema_pattern": "string",
	"output":         "string",
	"verbose":        "bool",
	"workers":        "int",
	"no_progress":    "bool",
}

func validateKey(key string) error {
	if _, ok := validKeys[key]; !ok {
		return fmt.Errorf("unknown config key: %q", key)
	}
	return nil
}

func coerceValue(key, raw string) (any, error) {
	switch validKeys[key] {
	case "bool":
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("key %q expects a bool (true/false), got %q", key, raw)
		}
		return v, nil
	case "int":
		v, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("key %q expects an integer, got %q", key, raw)
		}
		return v, nil
	default:
		return raw, nil
	}
}

// globalConfigPath returns the path to the global config file.
func globalConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "nodeval", ".nodeval.yaml"), nil
}

// readConfigFile reads a YAML config file into a map.
// Returns an empty map (no error) if the file does not exist.
func readConfigFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

// writeConfigFile marshals m to YAML and writes it to path,
// creating parent directories as needed.
func writeConfigFile(path string, m map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
```

**Step 4 : Vérifier que les tests passent**

```bash
go test ./cmd/... -run "TestReadWrite|TestValidate|TestCoerce" -v
```

Attendu : tous `PASS`

**Step 5 : Commit**

```bash
git add cmd/config_rw.go cmd/config_rw_test.go
git commit -m "feat: add config YAML read/write helpers"
```

---

### Task 2 : `config set`

**Files:**
- Modify: `cmd/config.go`
- Test: `cmd/config_set_test.go`

**Step 1 : Écrire les tests qui échouent**

Créer `cmd/config_set_test.go` :

```go
package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestConfigSet_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")

	if err := runConfigSet(path, "schemas", "./data"); err != nil {
		t.Fatal(err)
	}

	m, _ := readConfigFile(path)
	if m["schemas"] != "./data" {
		t.Errorf("expected schemas=./data, got %v", m["schemas"])
	}
}

func TestConfigSet_OverwritesValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	_ = writeConfigFile(path, map[string]any{"schemas": "."})

	if err := runConfigSet(path, "schemas", "./new"); err != nil {
		t.Fatal(err)
	}

	m, _ := readConfigFile(path)
	if m["schemas"] != "./new" {
		t.Errorf("expected schemas=./new, got %v", m["schemas"])
	}
}

func TestConfigSet_TypeCoercion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")

	if err := runConfigSet(path, "workers", "8"); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	var m map[string]any
	_ = yaml.Unmarshal(data, &m)
	if m["workers"] != 8 {
		t.Errorf("expected int 8, got %T %v", m["workers"], m["workers"])
	}
}

func TestConfigSet_UnknownKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	if err := runConfigSet(path, "badkey", "value"); err == nil {
		t.Error("expected error for unknown key")
	}
}
```

**Step 2 : Vérifier que les tests échouent**

```bash
go test ./cmd/... -run "TestConfigSet" -v
```

Attendu : erreur de compilation (`runConfigSet` non défini)

**Step 3 : Implémenter**

Dans `cmd/config.go`, ajouter après `configShowCmd` :

```go
// runConfigSet is the testable core of configSetCmd.
func runConfigSet(path, key, value string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	coerced, err := coerceValue(key, value)
	if err != nil {
		return err
	}
	m, err := readConfigFile(path)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}
	m[key] = coerced
	return writeConfigFile(path, m)
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration key in the local or global config file.

Valid keys: schemas, schema_pattern, output, verbose, workers, no_progress

Examples:
  nodeval config set schemas ./schemas
  nodeval config set output json
  nodeval config set --global workers 8`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		global, _ := cmd.Flags().GetBool("global")
		path, err := resolveConfigPath(global)
		if err != nil {
			return err
		}
		if err := runConfigSet(path, args[0], args[1]); err != nil {
			return err
		}
		color.Green("✅ %s = %s (in %s)", args[0], args[1], path)
		return nil
	},
}
```

Ajouter aussi le helper `resolveConfigPath` (utilisé par set, get, unset) :

```go
func resolveConfigPath(global bool) (string, error) {
	if global {
		return globalConfigPath()
	}
	return ".nodeval.yaml", nil
}
```

Dans `init()`, ajouter :

```go
configCmd.AddCommand(configSetCmd)
configSetCmd.Flags().Bool("global", false, "Write to global config (~/.config/nodeval/.nodeval.yaml)")
```

**Step 4 : Vérifier que les tests passent**

```bash
go test ./cmd/... -run "TestConfigSet" -v
```

Attendu : tous `PASS`

**Step 5 : Commit**

```bash
git add cmd/config.go cmd/config_set_test.go
git commit -m "feat: add config set command"
```

---

### Task 3 : `config get`

**Files:**
- Modify: `cmd/config.go`
- Test: `cmd/config_get_test.go`

**Step 1 : Écrire les tests qui échouent**

Créer `cmd/config_get_test.go` :

```go
package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func execConfigGet(key string) (string, error) {
	viper.Reset()
	viper.SetDefault("schemas", ".")
	viper.SetDefault("output", "terminal")
	viper.SetDefault("workers", 0)
	viper.SetDefault("verbose", false)
	viper.SetDefault("no_progress", false)
	viper.SetDefault("schema_pattern", "json-schema-Node_{type}.json")

	buf := &bytes.Buffer{}
	root := &cobra.Command{Use: "nodeval"}
	parent := &cobra.Command{Use: "config"}
	child := &cobra.Command{
		Use:  "get <key>",
		Args: cobra.ExactArgs(1),
		RunE: configGetCmd.RunE,
	}
	root.AddCommand(parent)
	parent.AddCommand(child)
	root.SetOut(buf)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"config", "get", key})
	err := root.Execute()
	return buf.String(), err
}

func TestConfigGet_Default(t *testing.T) {
	out, err := execConfigGet("output")
	if err != nil {
		t.Fatal(err)
	}
	if out != "terminal\n" {
		t.Errorf("expected 'terminal\\n', got %q", out)
	}
}

func TestConfigGet_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	_ = writeConfigFile(path, map[string]any{"output": "json"})

	viper.Reset()
	viper.SetDefault("output", "terminal")
	viper.SetConfigFile(path)
	_ = viper.ReadInConfig()

	out, err := execConfigGet("output")
	if err != nil {
		t.Fatal(err)
	}
	if out != "json\n" {
		t.Errorf("expected 'json\\n', got %q", out)
	}
}

func TestConfigGet_UnknownKey(t *testing.T) {
	_, err := execConfigGet("badkey")
	if err == nil {
		t.Error("expected error for unknown key")
	}
}
```

**Step 2 : Vérifier que les tests échouent**

```bash
go test ./cmd/... -run "TestConfigGet" -v
```

Attendu : erreur de compilation (`configGetCmd` non défini)

**Step 3 : Implémenter**

Dans `cmd/config.go`, ajouter :

```go
var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get the effective value of a configuration key",
	Long: `Print the effective value of a config key (CLI flags > local config > global config > defaults).

Valid keys: schemas, schema_pattern, output, verbose, workers, no_progress

Examples:
  nodeval config get schemas
  nodeval config get output`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		if err := validateKey(key); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), viper.Get(key))
		return nil
	},
}
```

Dans `init()`, ajouter :

```go
configCmd.AddCommand(configGetCmd)
```

**Step 4 : Vérifier que les tests passent**

```bash
go test ./cmd/... -run "TestConfigGet" -v
```

Attendu : tous `PASS`

**Step 5 : Commit**

```bash
git add cmd/config.go cmd/config_get_test.go
git commit -m "feat: add config get command"
```

---

### Task 4 : `config unset`

**Files:**
- Modify: `cmd/config.go`
- Test: `cmd/config_unset_test.go`

**Step 1 : Écrire les tests qui échouent**

Créer `cmd/config_unset_test.go` :

```go
package cmd

import (
	"testing"
	"path/filepath"

	"github.com/fatih/color"
)

func init() {
	color.NoColor = true
}

func TestConfigUnset_RemovesKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	_ = writeConfigFile(path, map[string]any{"schemas": "./data", "output": "json"})

	if err := runConfigUnset(path, "output"); err != nil {
		t.Fatal(err)
	}

	m, _ := readConfigFile(path)
	if _, ok := m["output"]; ok {
		t.Error("expected output to be removed")
	}
	if m["schemas"] != "./data" {
		t.Error("expected schemas to remain")
	}
}

func TestConfigUnset_AbsentKey_NoError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	_ = writeConfigFile(path, map[string]any{"schemas": "."})

	if err := runConfigUnset(path, "output"); err != nil {
		t.Errorf("expected no error for absent key, got: %v", err)
	}
}

func TestConfigUnset_MissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	if err := runConfigUnset(path, "schemas"); err == nil {
		t.Error("expected error when file does not exist")
	}
}

func TestConfigUnset_UnknownKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".nodeval.yaml")
	_ = writeConfigFile(path, map[string]any{"schemas": "."})
	if err := runConfigUnset(path, "badkey"); err == nil {
		t.Error("expected error for unknown key")
	}
}
```

**Step 2 : Vérifier que les tests échouent**

```bash
go test ./cmd/... -run "TestConfigUnset" -v
```

Attendu : erreur de compilation (`runConfigUnset` non défini)

**Step 3 : Implémenter**

Dans `cmd/config.go`, ajouter :

```go
// runConfigUnset is the testable core of configUnsetCmd.
func runConfigUnset(path, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", path)
	}
	m, err := readConfigFile(path)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}
	if _, ok := m[key]; !ok {
		color.Yellow("⚠️  key %q not set in %s", key, path)
		return nil
	}
	delete(m, key)
	return writeConfigFile(path, m)
}

var configUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Remove a configuration key",
	Long: `Remove a key from the local or global config file.
If the key is not present, a warning is printed and the command exits successfully.

Valid keys: schemas, schema_pattern, output, verbose, workers, no_progress

Examples:
  nodeval config unset verbose
  nodeval config unset --global output`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		global, _ := cmd.Flags().GetBool("global")
		path, err := resolveConfigPath(global)
		if err != nil {
			return err
		}
		if err := runConfigUnset(path, args[0]); err != nil {
			return err
		}
		return nil
	},
}
```

Dans `init()`, ajouter :

```go
configCmd.AddCommand(configUnsetCmd)
configUnsetCmd.Flags().Bool("global", false, "Write to global config (~/.config/nodeval/.nodeval.yaml)")
```

**Step 4 : Vérifier que tous les tests passent**

```bash
go test ./cmd/... -v
```

Attendu : tous `PASS`

**Step 5 : Commit**

```bash
git add cmd/config.go cmd/config_unset_test.go
git commit -m "feat: add config unset command"
```

---

### Task 5 : README

**Files:**
- Modify: `README.md`

**Step 1 : Ajouter la section `config set/get/unset`**

Dans `README.md`, après la section `### nodeval config show`, ajouter :

```markdown
### `nodeval config set <key> <value>`

Sets a configuration key in the local `.nodeval.yaml` or the global `~/.config/nodeval/.nodeval.yaml`.

```bash
nodeval config set <key> <value> [--global]
```

| Flag       | Default | Description                        |
| ---------- | ------- | ---------------------------------- |
| `--global` | `false` | Write to the global config file.   |

Valid keys: `schemas`, `schema_pattern`, `output`, `verbose`, `workers`, `no_progress`

#### **Examples**

```bash
nodeval config set schemas ./schemas
nodeval config set output json
nodeval config set --global workers 8
```

---

### `nodeval config get <key>`

Prints the effective value of a config key (CLI flags > local config > global config > defaults).

```bash
nodeval config get <key>
```

#### **Examples**

```bash
nodeval config get schemas
nodeval config get output
```

---

### `nodeval config unset <key>`

Removes a key from the local or global config file. Exits successfully with a warning if the key is not set.

```bash
nodeval config unset <key> [--global]
```

| Flag       | Default | Description                        |
| ---------- | ------- | ---------------------------------- |
| `--global` | `false` | Write to the global config file.   |

#### **Examples**

```bash
nodeval config unset verbose
nodeval config unset --global output
```
```

**Step 2 : Vérifier le build final**

```bash
go build ./... && go test ./...
```

Attendu : tous `PASS`

**Step 3 : Commit**

```bash
git add README.md
git commit -m "docs: document config set/get/unset commands"
```
