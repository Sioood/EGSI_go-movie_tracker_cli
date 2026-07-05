# MovieTracker CLI

Application terminal en Go pour suivre ses films. Projet en cours — voir [PLAN.md](PLAN.md).

## Prérequis

- Go 1.22+

## Phase 2 — TUI et navigation

```bash
make build
make test
make run-cli    # migre la DB client, lance la TUI
make run-server # migre la DB serveur, affiche un log
```

Les bases SQLite sont créées dans `data/client.db` et `data/server.db`.

La couche locale contient maintenant un repository SQLite pour les films, un repository pour les entrées de visionnage et un `MovieService` chargé des validations métier.
La CLI lance aussi une TUI Bubble Tea avec navigation entre les écrans Splash, Menu, Films, Détail, Statistiques, Paramètres, Connexion et Aide.

## Structure

```
cmd/cli/          # Binaire TUI (squelette Phase 0)
cmd/server/       # Binaire API (squelette Phase 0)
internal/domain/  # Entités métier
internal/apperrors/
internal/database/
internal/logging/
internal/repository/
internal/service/
internal/tui/
migrations/
```
