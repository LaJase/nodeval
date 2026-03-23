# Design : Amélioration des descriptions d'erreurs

**Date** : 2026-03-23
**Statut** : Approuvé

## Contexte

Actuellement, les modes `--verbose` et normal affichent les erreurs avec le même format (une ligne par erreur). De plus, `extractError` ne remonte qu'une seule erreur par fichier, même si plusieurs existent.

## Objectif

- **Mode normal** : une ligne par fichier ; si 1 erreur → format actuel (`path : message`) ; si N>1 erreurs → `N errors` (sans extraire les détails, pour ne pas dégrader les perfs)
- **Mode verbose** : toutes les erreurs par fichier, regroupées et indentées sous le nom du fichier

## Modèle de données

Ajout d'un champ `Count` à `FileError` (`validator.go`) :

```go
type FileError struct {
    File    string `json:"file"`
    Path    string `json:"path,omitempty"`
    Message string `json:"message,omitempty"`
    Count   int    `json:"count,omitempty"` // >1 en mode normal si plusieurs erreurs
}
```

- `Count > 1` : plusieurs erreurs non extraites (mode normal)
- `Count == 0` ou `Count == 1` : Path/Message présents

## Changements dans `validator.go`

### Nouvelles fonctions

**`countLeafErrors(ve)`** — zéro alloc, mode normal :
```go
func countLeafErrors(ve *jsonschema.ValidationError) int {
    if len(ve.Causes) == 0 {
        return 1
    }
    n := 0
    for _, c := range ve.Causes {
        n += countLeafErrors(c)
    }
    return n
}
```

**`extractAllErrors(ve)`** — mode verbose, toutes les feuilles :
```go
func extractAllErrors(ve *jsonschema.ValidationError) []FileError
```

### `validateFile` mis à jour

Signature : `validateFile(sch, fPath string, verbose bool) ([]FileError, bool)`

| Cas | Retour |
|-----|--------|
| Pas d'erreur | `[], true` |
| Normal + 1 erreur | `[{File, Path, Message}], false` |
| Normal + N>1 erreurs | `[{File, Count: N}], false` |
| Verbose + erreur(s) | `[{File, Path, Message}, ...], false` (toutes les feuilles) |

Le worker transmet `opts.Verbose` à `validateFile`. `localBatch.add` accepte `[]FileError`.

## Affichage dans `terminal.go`

**Mode normal :**
```
❌ foo_M.json : data.name : expected string, but got number
❌ bar_M.json : 3 errors
```

**Mode verbose :**
```
❌ foo_M.json :
   data.name : expected string, but got number
   data.age  : minimum: value must be >= 0

❌ bar_M.json :
   (root) : missing required fields: id, label
```

Les `FileError` sont regroupés par `File` (ordre de première apparition) dans `Render`.

## Impact sur les autres reporters

- **JSON** : `count` apparaît dans le output uniquement quand `> 0` (`omitempty`) — pas de breaking change
- **JUnit** : aucun changement
