# Design : jsnsch CLI

**Date :** 2026-03-21
**Statut :** Validé

## Contexte

`jsnsch` est un validateur JSON schema multithreadé. La base existante (`main.go`) est fonctionnelle mais n'a pas d'interface CLI structurée. L'objectif est de la transformer en un outil CLI professionnel, maintenable, et extensible.

## Commandes

```
jsnsch
├── validate <directory>
│   ├── --schemas <dir>              Dossier des schémas (défaut: .)
│   ├── --types M,R,I                Types à valider (défaut: auto-detect)
│   ├── --all                        Tous les types détectés
│   ├── --output terminal|json|junit Format de sortie (défaut: terminal)
│   ├── --verbose                    Erreurs détaillées (arbre complet)
│   ├── --workers N                  Nb de workers (défaut: NumCPU)
│   └── --no-progress                Désactive les barres de progression
│
├── schema
│   ├── list                         Liste les schémas détectés
│   └── check <type>                 Vérifie qu'un schéma est valide
│
└── config
    ├── init                         Génère un .jsnsch.yaml exemple
    └── show                         Affiche la config active
```

## Architecture interne

```
jsnsch/
├── main.go
├── cmd/
│   ├── root.go       # Cobra root + Viper init
│   ├── validate.go   # Commande validate
│   ├── schema.go     # Commandes schema list/check
│   └── config.go     # Commandes config init/show
├── internal/
│   ├── config/
│   │   └── config.go    # Struct Config + chargement Viper
│   ├── scanner/
│   │   └── scanner.go   # WalkDir + détection auto des types
│   ├── validator/
│   │   └── validator.go # Worker pool
│   ├── schema/
│   │   ├── loader.go    # Interface Loader + LocalLoader
│   │   └── detect.go    # Détection des types depuis les schémas présents
│   └── reporter/
│       ├── terminal.go  # Sortie couleur + barres mpb
│       ├── json.go      # Sortie JSON structuré
│       └── junit.go     # Sortie JUnit XML
├── .jsnsch.yaml
└── go.mod
```

## Extensibilité schémas

Interface `schema.Loader` pour permettre l'ajout futur de sources distantes (zip/tar depuis URL) sans modifier le reste du code :

```go
type Loader interface {
    Load(typeNode string) (*jsonschema.Schema, error)
}
```

Implémentations prévues : `LocalLoader` (maintenant), `URLLoader` / `ArchiveLoader` (plus tard).

## Fichier de config `.jsnsch.yaml`

```yaml
schemas: ./schemas
workers: 0        # 0 = auto (NumCPU)
output: terminal  # terminal | json | junit
verbose: false
types:            # optionnel, sinon auto-détecté
  - M
  - R
```

Priorité : flag CLI > `.jsnsch.yaml` local > `~/.config/jsnsch/config.yaml` > valeurs par défaut

## Formats de sortie

**`--output json`**
```json
{
  "duration_ms": 1243,
  "total": 1500,
  "errors": 3,
  "success": true,
  "results": [
    {
      "type": "M",
      "success": 498,
      "errors": 2,
      "details": [
        { "file": "node_M.json", "path": "root > address", "message": "street is required" }
      ]
    }
  ]
}
```

**`--output junit`**
```xml
<testsuites>
  <testsuite name="Type M" tests="500" failures="2">
    <testcase name="node_M.json">
      <failure>root > address : street is required</failure>
    </testcase>
  </testsuite>
</testsuites>
```

## Codes de sortie

| Code | Signification |
|------|---------------|
| 0    | Tous les fichiers valides |
| 1    | Au moins une erreur de validation |
| 2    | Erreur de configuration / schéma manquant |
| 3    | Erreur I/O (dossier inexistant, permission refusée) |

## Cross-platform

- Chemins via `filepath.Join` — pas de séparateurs hardcodés
- Couleurs désactivées automatiquement si pas de TTY (`fatih/color` + `go-isatty`)
- `--no-progress` pour CI/CD sans TTY
- Cibles de build : `linux/amd64`, `linux/arm64`, `windows/amd64`

## Convention de nommage des fichiers

Pattern : `*_<TYPE>.json` (préfixe libre, suffixe `_<TYPE>.json`)
