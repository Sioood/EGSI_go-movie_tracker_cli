# MovieTracker CLI

Application terminal en Go pour suivre ses films. Projet en cours — voir [PLAN.md](PLAN.md).

## Prérequis

- Go 1.22+

## Phase 4 — Recherche et filtres

```bash
make build
make test
make run-cli    # migre la DB client, lance la TUI
make run-server # migre la DB serveur, affiche un log
```

Les bases SQLite sont créées dans `data/client.db` et `data/server.db`.

La couche locale contient maintenant un repository SQLite pour les films, un repository pour les entrées de visionnage et un `MovieService` chargé des validations métier.
La CLI lance aussi une TUI Bubble Tea avec navigation entre les écrans Splash, Menu, Films, Détail, Statistiques, Paramètres, Connexion et Aide.
Depuis l'écran Films, il est possible d'ajouter un film, ouvrir son détail, le marquer vu, renseigner une date, une note et une critique, puis sauvegarder le tout en SQLite local.
L'écran Films propose aussi une recherche temps réel par titre, des filtres vus/non vus/notés/sans note et un tri par titre/date/note.

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
