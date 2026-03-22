# Config set/get/unset — Design

## Goal

Add `nodeval config set`, `nodeval config get`, and `nodeval config unset` commands following the `git config` model.

## Commands

```bash
nodeval config set <key> <value>           # write to local .nodeval.yaml
nodeval config set --global <key> <value>  # write to ~/.config/nodeval/.nodeval.yaml
nodeval config get <key>                   # read effective value (Viper merge)
nodeval config unset <key>                 # remove key from local config
nodeval config unset --global <key>        # remove key from global config
```

## Architecture

- Three new sub-commands under `configCmd`: `configSetCmd`, `configGetCmd`, `configUnsetCmd`
- `--global` flag on `set` and `unset`; `get` always reads the merged Viper value
- Cobra `Long` and `Example` fields set on each command for complete `--help` output
- Valid keys: `schemas`, `schema_pattern`, `output`, `verbose`, `workers`, `no_progress` — unknown keys rejected with exit 3

## Data flow

**set / unset:**
1. Resolve target path: `--global` → `~/.config/nodeval/.nodeval.yaml`, else `.nodeval.yaml` in CWD
2. `set`: create file if absent; `unset`: error if file absent
3. Read file → `yaml.Unmarshal` → `map[string]any`
4. Modify map (`set`: add/overwrite, `unset`: delete key)
5. `yaml.Marshal` → write file back

**get:**
1. Read via `viper.Get(key)` (already merged by Viper)
2. Print raw value to stdout

## Error handling

| Situation | Behaviour |
|-----------|-----------|
| Unknown key | Exit 3, message `unknown config key: "foo"` |
| File not readable/writable | Exit 3 |
| `unset` on absent key | Yellow warning, exit 0 |
| `get` with no explicit value | Shows Viper default |

## Dependencies

`gopkg.in/yaml.v3` — already pulled transitively by Viper, no new `go.mod` entry needed.

## Tests

- `set` local: creates file when absent, overwrites existing value
- `set --global`: writes to global path
- `unset`: removes key; no-op (exit 0) when key absent
- `get`: returns effective value respecting local > global > default priority
- Invalid key: returns expected error
