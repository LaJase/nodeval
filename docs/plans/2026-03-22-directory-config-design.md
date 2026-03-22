# Directory Config Option — Design

## Goal

Add `directory` as a configurable key so `nodeval validate` can be run without a positional argument when the directory is set in `.nodeval.yaml` or the global config.

## Approach

Make the positional `<directory>` argument optional. CLI argument always takes priority over config. Backward compatible.

## Changes

| File | Change |
|------|--------|
| `internal/config/config.go` | Add `mapstructure:"directory"` tag to `Directory` field |
| `cmd/config_rw.go` | Add `"directory": "string"` to `validKeys` |
| `cmd/root.go` | Add `viper.SetDefault("directory", "")` |
| `cmd/config.go` | Add `directory` to `configShowCmd`, `configInitCmd` template, and `configSetCmd` Long |
| `cmd/validate.go` | `cobra.MaximumNArgs(1)` + fallback to `viper.GetString("directory")` |
| `README.md` | Document `directory` config key and updated validate usage |

## Data flow

1. User passes `nodeval validate ./data --all` → `args[0]` used directly
2. User runs `nodeval validate --all` → reads `viper.GetString("directory")`
3. Neither provided → error: `"no directory specified — pass it as argument or set 'directory' in config"`

## Priority

CLI argument > local `.nodeval.yaml` > global config > error (no default)

## Tests

- Positional arg overrides config value
- Config value used when arg absent
- Error when neither arg nor config provided
- `nodeval config set directory ./data` stores and retrieves correctly
