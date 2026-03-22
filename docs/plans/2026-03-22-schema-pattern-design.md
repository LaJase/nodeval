# Design : Schema pattern configurable

**Date :** 2026-03-22
**Statut :** Approuvé

---

## Contexte

Le pattern de nommage des fichiers schema est actuellement hardcodé à deux endroits :

- `internal/schema/detect.go` : `const schemaPrefix = "json-schema-Node_"`
- `internal/schema/loader.go` : `fmt.Sprintf("json-schema-Node_%s.json", typeNode)`

Cela empêche les utilisateurs d'utiliser leur propre convention de nommage sans modifier le code source.

---

## Objectif

Rendre le pattern de nommage des schemas configurable via :

1. Un flag CLI `--schema-pattern`
2. Une clé `schema_pattern` dans `.nodeval.yaml`
3. Avec la priorité habituelle : CLI > config > défaut

---

## Syntaxe du placeholder

Le placeholder `{type}` représente le nom du type dans le pattern.

```
json-schema-Node_{type}.json   →  json-schema-Node_M.json
schema_{type}.json             →  schema_M.json
{type}_node.json               →  M_node.json
```

La valeur par défaut est `json-schema-Node_{type}.json` (rétrocompatible).

---

## Architecture

### Parsing du pattern

Toute logique de parsing est centralisée dans `internal/schema` :

```
pattern → strings.SplitN(pattern, "{type}", 2) → [prefix, suffix]
```

Validation : si `{type}` est absent → erreur immédiate.

### Flux de données

```
CLI --schema-pattern / .nodeval.yaml schema_pattern
    ↓ Viper
viper.GetString("schema_pattern")
    ↓
schema.DetectTypes(dir, pattern)    // détection des types disponibles
schema.NewLocalLoader(dir, pattern) // chargement des schemas
```

### Fichiers modifiés

| Fichier | Changement |
|---------|-----------|
| `internal/config/config.go` | Ajout `SchemaPattern string` + défaut |
| `internal/schema/detect.go` | `DetectTypes(dir, pattern string)` |
| `internal/schema/loader.go` | `NewLocalLoader(dir, pattern string)` |
| `cmd/validate.go` | Flag `--schema-pattern` |
| `cmd/schema.go` | Flag `--schema-pattern` persistent, affichage dynamique |
| `cmd/root.go` | `viper.SetDefault("schema_pattern", ...)` |
| `cmd/config.go` | Affiche `schema_pattern` dans `config show` |

---

## Gestion d'erreurs

- Pattern sans `{type}` → erreur explicite au démarrage
- Aucun changement de comportement pour les utilisateurs existants (défaut identique)

---

## Tests

| Test | Fichier |
|------|---------|
| `TestDetectTypes_CustomPattern` | `detect_test.go` |
| `TestDetectTypes_InvalidPattern` | `detect_test.go` |
| `TestLocalLoader_CustomPattern` | `loader_test.go` |
| `TestLocalLoader_InvalidPattern` | `loader_test.go` |

Les tests existants continuent de passer sans modification.
