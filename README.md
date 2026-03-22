# nodeval

> Multithreaded JSON Schema validator with auto-detection, multi-format output, and a Cobra CLI.

---

## Overview

`nodeval` recursively scans a directory for JSON files, automatically detects which schemas apply to them based on a
naming convention, and validates every file in parallel using a configurable worker pool.

### **Key features**

- **Multithreaded** — worker pool sized to `NumCPU` by default; fully configurable.
- **Auto-detection** — scans the schema directory and discovers available types without any manual listing.
- **Multi-format output** — colored terminal report, machine-readable JSON, and JUnit XML for CI pipelines.
- **Cobra CLI** — structured sub-command tree (`validate`, `schema`, `config`) with persistent flags and Viper-backed
  configuration.
- **Priority layering** — CLI flags override local config, which overrides global config, which overrides built-in
  defaults.

---

## Installation

Requirements: **Go 1.25+**

Clone the repository then compile for your platform. The resulting binary is fully self-contained — no runtime
dependencies, no installer.

### Linux / macOS

```bash
git clone <repo-url> nodeval
cd nodeval
CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o nodeval .
```

### Windows (PowerShell)

```powershell
git clone <repo-url> nodeval
cd nodeval
$env:CGO_ENABLED=0; go build -ldflags="-s -w" -trimpath -o nodeval.exe .
```

> **Flags explained**
>
> - `CGO_ENABLED=0` — fully static binary, no external DLL dependency
> - `-ldflags="-s -w"` — strips debug symbols, reduces binary size
> - `-trimpath` — removes local file paths from the binary

---

## Quick Start

```bash
# Validate all detected types in ./data using schemas in the current directory
nodeval validate ./data --all

# Validate only types M and R, show full error detail
nodeval validate ./data --types M,R --verbose

# Export results as JSON (suitable for jq / downstream tooling)
nodeval validate ./data --all --output json > results.json

# Export results as JUnit XML (suitable for CI test reporters)
nodeval validate ./data --all --output junit > results.xml
```

---

## Commands

### `nodeval validate [directory]`

Recursively scans `[directory]` for files matching `*_<TYPE>.json` and validates each one against the corresponding
`json-schema-Node_<TYPE>.json` schema.

```bash
nodeval validate [directory] [flags]
```

| Flag                     | Default          | Description                                                       |
| ------------------------ | ---------------- | ----------------------------------------------------------------- |
| `--schemas <dir>`        | `.`              | Directory containing the JSON schema files.                       |
| `--schema-pattern <pat>` | _(config value)_ | Schema filename pattern. Use `{type}` as placeholder.             |
| `--types <list>`         | _(auto)_         | Types to validate (e.g. `M,R,I`). Auto-detected when omitted.     |
| `--all`                  | `false`          | Validate all auto-detected types, ignoring `--types`.             |
| `--output <fmt>`         | `terminal`       | Output format: `terminal`, `json`, or `junit`.                    |
| `--verbose`              | `false`          | Print the full JSON path and message for each invalid file.       |
| `--workers <n>`          | `0`              | Number of parallel workers. `0` means one worker per logical CPU. |
| `--no-progress`          | `false`          | Suppress the per-type progress bars (recommended in CI/CD).       |

#### **Examples**

```bash
# Auto-detect types, schemas alongside the data
nodeval validate ./data --all

# Use a dedicated schemas folder with 4 workers
nodeval validate ./data --all --schemas ./schemas --workers 4

# Validate a specific subset of types, disable progress bars
nodeval validate ./data --types M,R --no-progress

# Machine-readable output redirected to a file
nodeval validate ./data --all --output json > results.json

# JUnit XML for GitLab / Jenkins test reports
nodeval validate ./data --all --output junit --no-progress > results.xml

# Full error detail in the terminal
nodeval validate ./data --types I --verbose

# Directory set in config, run without argument
nodeval validate --all
```

---

### `nodeval schema list`

Lists every schema detected in the `--schemas` directory and shows the type name it corresponds to.

```bash
nodeval schema list --schemas ./schemas
```

| Flag                     | Default          | Description                                           |
| ------------------------ | ---------------- | ----------------------------------------------------- |
| `--schemas <dir>`        | `.`              | Directory to inspect for schema files.                |
| `--schema-pattern <pat>` | _(config value)_ | Schema filename pattern. Use `{type}` as placeholder. |

#### **Example output**

```text
Schemas detected in ./schemas:
  ✓ Type I  → json-schema-Node_I.json
  ✓ Type M  → json-schema-Node_M.json
  ✓ Type R  → json-schema-Node_R.json
```

---

### `nodeval schema check [type...]`

Loads and parses one or more schemas to verify they are valid and accessible. Pass explicit type names, or use `--all`
to check every schema auto-detected in the `--schemas` directory. Useful for debugging schema issues before running a
full validation.

```bash
nodeval schema check [type...] [flags]
```

| Flag                     | Default          | Description                                             |
| ------------------------ | ---------------- | ------------------------------------------------------- |
| `--schemas <dir>`        | `.`              | Directory containing the schema file.                   |
| `--schema-pattern <pat>` | _(config value)_ | Schema filename pattern. Use `{type}` as placeholder.   |
| `--all`                  | `false`          | Check all auto-detected types in the schemas directory. |

Returns exit code `0` on success, `2` on failure (missing or invalid schema).

#### **Examples**

```bash
# Check a single schema
nodeval schema check M --schemas ./schemas

# Check multiple schemas at once
nodeval schema check M R --schemas ./schemas

# Check every schema auto-detected in the schemas directory
nodeval schema check --all --schemas ./schemas
```

---

### `nodeval config init`

Generates a commented `.nodeval.yaml` template in the current working directory. Aborts safely if the file already
exists.

```bash
nodeval config init
```

---

### `nodeval config show`

Prints the effective configuration after merging CLI flags, the loaded config file, and built-in defaults. Useful to
confirm which config file is active and what values are in effect.

```bash
nodeval config show
```

#### **Config show output**

```text
Config file : ./.nodeval.yaml

  schemas        : ./schemas
  schema-pattern : json-schema-Node_{type}.json
  types          : []
  output         : terminal
  verbose        : false
  workers        : 0
  no-progress    : false
```

---

### `nodeval config set <key> <value>`

Sets a configuration key in the local `.nodeval.yaml` or the global `~/.config/nodeval/.nodeval.yaml`.

```bash
nodeval config set <key> <value> [--global]
```

| Flag       | Default | Description                      |
| ---------- | ------- | -------------------------------- |
| `--global` | `false` | Write to the global config file. |

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

Valid keys: `schemas`, `schema_pattern`, `output`, `verbose`, `workers`, `no_progress`

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

| Flag       | Default | Description                      |
| ---------- | ------- | -------------------------------- |
| `--global` | `false` | Write to the global config file. |

Valid keys: `schemas`, `schema_pattern`, `output`, `verbose`, `workers`, `no_progress`

#### **Examples**

```bash
nodeval config unset verbose
nodeval config unset --global output
```

---

## Configuration

`nodeval` uses [Viper](https://github.com/spf13/viper) for configuration management.

### Priority order (highest to lowest)

1. **CLI flags** — flags passed directly on the command line always win.
2. **Local config** — `.nodeval.yaml` in the current working directory.
3. **Global config** — `~/.config/nodeval/.nodeval.yaml`.
4. **Built-in defaults** — see the table below.

### `.nodeval.yaml` — full example

Generate this file with `nodeval config init`, then edit as needed.

```yaml
# nodeval configuration

# Directory containing the JSON files to validate.
# When set, nodeval validate can be run without a positional argument.
# directory: ./data

# Directory containing the JSON schema files
# Default: . (current directory)
schemas: ./schemas

# Schema filename pattern. Use {type} as placeholder for the type name.
# Default: json-schema-Node_{type}.json
# schema_pattern: json-schema-Node_{type}.json

# Types to validate. Leave commented out for automatic detection from the
# schemas directory. Explicit list takes effect when --all is not used.
# types:
#   - M
#   - R
#   - I

# Output format: terminal | json | junit
# Default: terminal
output: terminal

# Print full JSON path + message for each validation error
# Default: false
verbose: false

# Number of parallel validation workers (0 = one per logical CPU)
# Default: 0
workers: 0

# Disable per-type progress bars — recommended for CI/CD environments
# Default: false
no_progress: false
```

| Key              | Type   | Default                        | Description                                           |
| ---------------- | ------ | ------------------------------ | ----------------------------------------------------- |
| `directory`      | string | _(empty)_                      | Default directory for JSON files to validate.         |
| `schemas`        | string | `.`                            | Path to the directory holding schema files.           |
| `schema_pattern` | string | `json-schema-Node_{type}.json` | Schema filename pattern (`{type}` = type name).       |
| `types`          | list   | _(empty)_                      | Explicit list of types; auto-detected when empty.     |
| `output`         | string | `terminal`                     | Report format: `terminal`, `json`, or `junit`.        |
| `verbose`        | bool   | `false`                        | Include full JSON path and message in error output.   |
| `workers`        | int    | `0`                            | Worker pool size; `0` defaults to `runtime.NumCPU()`. |
| `no_progress`    | bool   | `false`                        | Suppress progress bars.                               |

---

## Schema naming convention

`nodeval` relies on a strict naming convention to associate data files with their schemas automatically.

### Schema files

Schema files must be placed in the schemas directory (default `.`, overridable with `--schemas`). The filename pattern
defaults to `json-schema-Node_{type}.json` and can be changed with `--schema-pattern` or `schema_pattern` in
`.nodeval.yaml`.

Default pattern:

```text
json-schema-Node_<TYPE>.json
```

Examples:

```text
json-schema-Node_M.json
json-schema-Node_R.json
json-schema-Node_I.json
```

### Data files

Data files to be validated must follow the pattern:

```text
*_<TYPE>.json
```

The `<TYPE>` suffix (the part between the last `_` and `.json`) is matched against the available schemas. Examples:

```text
node_M.json          → validated against json-schema-Node_M.json
report_2024_R.json   → validated against json-schema-Node_R.json
config_I.json        → validated against json-schema-Node_I.json
```

Files whose suffix does not match any known type are silently skipped.

---

## Output formats

### terminal (default)

Colored, human-readable output with optional progress bars. The final summary groups results by type.

```text
----------------------------------------------------------------------------------------------------

❌ bad_file_M.json : root > SomeDefinition : field xyz is required

Summary:
> Nodes M  :  142 OK |  1 Errors
> Nodes R  :   87 OK |  0 Errors

----------------------------------------------------------------------------------------------------

⏱️ Total time : 312ms
🚨 TOTAL : 230 files analyzed | 1 errors (INVALID)
```

With `--verbose`, each error is expanded over three lines:

```text
❌ bad_file_M.json
    Path    : root > SomeDefinition
    Message : field xyz is required
```

### JSON

Emitted to stdout; redirect to a file or pipe to `jq`.

```json
{
  "duration_ms": 312,
  "total": 230,
  "errors": 1,
  "success": false,
  "results": [
    {
      "type": "M",
      "success": 142,
      "errors": 1,
      "details": [
        {
          "file": "bad_file_M.json",
          "path": "root > SomeDefinition",
          "message": "field xyz is required"
        }
      ]
    },
    {
      "type": "R",
      "success": 87,
      "errors": 0
    }
  ]
}
```

`details` is omitted from a type entry when there are no errors.

### JUnit XML

Compatible with GitLab CI, Jenkins, and any JUnit-aware test reporter.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<testsuites>
  <testsuite name="Type M" tests="143" failures="1">
    <testcase name="valid_file_1"></testcase>
    <!-- ... One <testcase> per valid file ... -->
    <testcase name="bad_file_M.json">
      <failure message="root &gt; SomeDefinition">field xyz is required</failure>
    </testcase>
  </testsuite>
  <testsuite name="Type R" tests="87" failures="0">
    <!-- ... -->
  </testsuite>
</testsuites>
```

---

## Exit codes

| Code | Meaning                                                                                        |
| ---- | ---------------------------------------------------------------------------------------------- |
| `0`  | All files are valid.                                                                           |
| `1`  | At least one file failed schema validation (`ValidationError`).                                |
| `2`  | Configuration or schema problem — e.g. a schema file is missing or unreadable (`ConfigError`). |
| `3`  | Unexpected runtime error (I/O failure, bad arguments, etc.).                                   |

---

## Cross-platform builds

All commands use the same optimized flags. Run from the repository root.

### Linux

```bash
# amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o nodeval .

# arm64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o nodeval-arm64 .
```

### Windows — cross-compile (PowerShell)

```powershell
# amd64
$env:CGO_ENABLED=0; $env:GOOS="windows"; $env:GOARCH="amd64"
go build -ldflags="-s -w" -trimpath -o nodeval.exe .

# arm64
$env:CGO_ENABLED=0; $env:GOOS="windows"; $env:GOARCH="arm64"
go build -ldflags="-s -w" -trimpath -o nodeval-arm64.exe .
```

### macOS

```bash
# Intel
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o nodeval .

# Apple Silicon
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -trimpath -o nodeval-arm64 .
```

---

## Project structure

```text
nodeval/
├── main.go                              # Entry point — delegates to cmd.Execute()
├── go.mod                               # Module declaration and dependencies
├── go.sum                               # Dependency checksums
│
├── cmd/
│   ├── root.go                          # Root command, config initialisation, exit-code mapping
│   ├── validate.go                      # `nodeval validate` command and all its flags
│   ├── schema.go                        # `nodeval schema list` and `nodeval schema check`
│   ├── config.go                        # nodeval config init, show, set, get, unset
│   └── config_rw.go                     # YAML read/write helpers for config set/get/unset
│
├── internal/
│   ├── config/
│   │   ├── config.go                    # Config struct, Default() values, LoadFrom() helper
│   │   └── config_test.go               # Unit tests for config loading
│   │
│   ├── reporter/
│   │   ├── reporter.go                  # Reporter interface and Report type
│   │   ├── terminal.go                  # Colored terminal renderer
│   │   ├── json.go                      # JSON renderer
│   │   ├── json_test.go                 # Unit tests for JSON output
│   │   ├── junit.go                     # JUnit XML renderer
│   │   └── junit_test.go                # Unit tests for JUnit output
│   │
│   ├── scanner/
│   │   ├── scanner.go                   # Recursive directory scan; groups files by type
│   │   └── scanner_test.go              # Unit tests for the scanner
│   │
│   ├── schema/
│   │   ├── detect.go                    # DetectTypes() — discovers types from schema filenames
│   │   ├── detect_test.go               # Unit tests for type detection
│   │   ├── loader.go                    # NewLocalLoader() — loads and compiles JSON schemas
│   │   └── loader_test.go               # Unit tests for schema loading
│   │
│   └── validator/
│       ├── validator.go                 # Run() — parallel worker pool, per-file validation logic
│       └── validator_test.go            # Integration tests for the validator
```
