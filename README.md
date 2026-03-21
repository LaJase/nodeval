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

Requirements: Go 1.22+ (the module declares `go 1.26.1`).

### **Linux**

```bash
git clone <repo-url> nodeval
cd nodeval
go build -o nodeval .
```

### **Windows (amd64)**

```powershell
$env:GOOS="windows"; $env:GOARCH="amd64"
go build -o nodeval.exe .
```

### **Windows (arm64)**

```powershell
$env:GOOS="windows"; $env:GOARCH="arm64"
go build -o nodeval.exe .
```

The resulting binary is self-contained — no runtime dependencies.

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

### `nodeval validate <directory>`

Recursively scans `<directory>` for files matching `*_<TYPE>.json` and validates each one against the corresponding
`json-schema-Node_<TYPE>.json` schema.

```bash
nodeval validate <directory> [flags]
```

| Flag              | Default    | Description                                                                                                            |
| ----------------- | ---------- | ---------------------------------------------------------------------------------------------------------------------- |
| `--schemas <dir>` | `.`        | Directory containing the JSON schema files.                                                                            |
| `--types <list>`  | _(auto)_   | Comma-separated list of types to validate (e.g. `M,R,I`). When omitted the schemas directory is scanned automatically. |
| `--all`           | `false`    | Force validation of all auto-detected types, ignoring `--types`.                                                       |
| `--output <fmt>`  | `terminal` | Output format: `terminal`, `json`, or `junit`.                                                                         |
| `--verbose`       | `false`    | Print the full JSON path and error message for each invalid file.                                                      |
| `--workers <n>`   | `0`        | Number of parallel workers. `0` means one worker per logical CPU.                                                      |
| `--no-progress`   | `false`    | Suppress the per-type progress bars (recommended in CI/CD).                                                            |

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
```

---

### `nodeval schema list`

Lists every schema detected in the `--schemas` directory and shows the type name it corresponds to.

```bash
nodeval schema list --schemas ./schemas
```

| Flag              | Default | Description                            |
| ----------------- | ------- | -------------------------------------- |
| `--schemas <dir>` | `.`     | Directory to inspect for schema files. |

#### **Example output**

```
Schemas detected in ./schemas:
  ✓ Type I  → json-schema-Node_I.json
  ✓ Type M  → json-schema-Node_M.json
  ✓ Type R  → json-schema-Node_R.json
```

---

### `nodeval schema check <type>`

Loads and parses a single schema to verify it is valid and accessible. Useful for debugging schema issues before running
a full validation.

```bash
nodeval schema check M --schemas ./schemas
```

| Flag              | Default | Description                           |
| ----------------- | ------- | ------------------------------------- |
| `--schemas <dir>` | `.`     | Directory containing the schema file. |

Returns exit code `0` on success, `3` on failure.

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

#### **Example output**

```
Config file : ./.nodeval.yaml

  schemas     : ./schemas
  types       : []
  output      : terminal
  verbose     : false
  workers     : 0
  no-progress : false
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

# Directory containing the JSON schema files (json-schema-Node_<TYPE>.json)
# Default: . (current directory)
schemas: ./schemas

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

| Key           | Type   | Default    | Description                                           |
| ------------- | ------ | ---------- | ----------------------------------------------------- |
| `schemas`     | string | `.`        | Path to the directory holding schema files.           |
| `types`       | list   | _(empty)_  | Explicit list of types; auto-detected when empty.     |
| `output`      | string | `terminal` | Report format: `terminal`, `json`, or `junit`.        |
| `verbose`     | bool   | `false`    | Include full JSON path and message in error output.   |
| `workers`     | int    | `0`        | Worker pool size; `0` defaults to `runtime.NumCPU()`. |
| `no_progress` | bool   | `false`    | Suppress progress bars.                               |

---

## Schema naming convention

`nodeval` relies on a strict naming convention to associate data files with their schemas automatically.

### Schema files

Schema files must be placed in the schemas directory (default `.`, overridable with `--schemas`) and named:

```
json-schema-Node_<TYPE>.json
```

Examples:

```
json-schema-Node_M.json
json-schema-Node_R.json
json-schema-Node_I.json
```

### Data files

Data files to be validated must follow the pattern:

```
*_<TYPE>.json
```

The `<TYPE>` suffix (the part between the last `_` and `.json`) is matched against the available schemas. Examples:

```
node_M.json          → validated against json-schema-Node_M.json
report_2024_R.json   → validated against json-schema-Node_R.json
config_I.json        → validated against json-schema-Node_I.json
```

Files whose suffix does not match any known type are silently skipped.

---

## Output formats

### terminal (default)

Colored, human-readable output with optional progress bars. The final summary groups results by type.

```
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

```
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
    <!-- ... one <testcase> per valid file ... -->
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

```bash
# Linux amd64 (native)
go build -o nodeval .

# Linux arm64
GOOS=linux GOARCH=arm64 go build -o nodeval-linux-arm64 .

# Windows amd64
GOOS=windows GOARCH=amd64 go build -o nodeval.exe .

# Windows arm64
GOOS=windows GOARCH=arm64 go build -o nodeval-arm64.exe .

# macOS amd64
GOOS=darwin GOARCH=amd64 go build -o nodeval-darwin .

# macOS arm64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o nodeval-darwin-arm64 .
```

---

## Project structure

```
nodeval/
├── main.go                              # Entry point — delegates to cmd.Execute()
├── go.mod                               # Module declaration and dependencies
├── go.sum                               # Dependency checksums
├── nodeval                               # Compiled Linux binary (not committed)
├── nodeval.exe                           # Compiled Windows binary (not committed)
│
├── cmd/
│   ├── root.go                          # Root command, config initialisation, exit-code mapping
│   ├── validate.go                      # `nodeval validate` command and all its flags
│   ├── schema.go                        # `nodeval schema list` and `nodeval schema check`
│   └── config.go                        # `nodeval config init` and `nodeval config show`
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
│       └── validator.go                 # Run() — parallel worker pool, per-file validation logic
│
└── docs/
    └── plans/                           # Design and implementation planning documents
```
