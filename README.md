# MovieTracker CLI

Application terminal en Go pour suivre ses films. Projet en cours — voir [PLAN.md](PLAN.md).

## Prérequis

- Go 1.22+

## Phase 0 — Fondations

```bash
make build
make test
make run-cli    # migre la DB client, affiche un log
make run-server # migre la DB serveur, affiche un log
```

Les bases SQLite sont créées dans `data/client.db` et `data/server.db`.

## Structure

```
cmd/cli/          # Binaire TUI (squelette Phase 0)
cmd/server/       # Binaire API (squelette Phase 0)
internal/domain/  # Entités métier
internal/apperrors/
internal/database/
internal/logging/
migrations/
```
