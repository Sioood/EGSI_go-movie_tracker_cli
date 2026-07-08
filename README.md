# MovieTracker v1.0

Application terminal en Go pour suivre ses films, avec synchronisation hybride local ↔ serveur. Voir [PLAN.md](PLAN.md) pour la feuille de route complète.

## Prérequis

- Go **1.26+**
- [golangci-lint](https://golangci-lint.run/) (pour `make lint`)
- Docker + Docker Compose (optionnel, pour le serveur API)

## Installation rapide

```bash
make build
make test
```

Binaires produits :

- `bin/movietracker` — TUI locale (Bubble Tea)
- `bin/movietracker-server` — API REST

## Utilisation CLI

```bash
make run-cli
```

Au premier lancement :

- La base SQLite locale est créée dans `data/client.db` (répertoire courant)
- La configuration utilisateur est stockée dans `~/.movietracker/config.yaml`
- La session serveur (tokens) est dans `~/.movietracker/session.yaml`

### Modes

| Mode | Description |
|------|-------------|
| **Hors ligne** | CRUD films, notes, critiques et stats en local uniquement |
| **Connecté** | Auth serveur + sync automatique (toutes les 30 s) et manuelle (`S`) |

Depuis le menu : **Films**, **Statistiques**, **Paramètres**, **Connexion**, **Aide** (`?`).

Raccourcis principaux : `/` recherche, `f`/`t`/`c` filtres, `a` ajouter, `S` synchroniser, `q` quitter.

## Serveur API

### Local

```bash
export JWT_SECRET="votre-secret-long-et-aleatoire"
make run-server
```

Le serveur écoute sur `:8080` par défaut.

### Docker

```bash
cp .env.example .env
# Éditer JWT_SECRET dans .env
make docker-up
curl http://localhost:8080/health
```

Arrêt : `make docker-down`

### Variables d'environnement

| Variable | Défaut | Description |
|----------|--------|-------------|
| `JWT_SECRET` | *(requis)* | Clé de signature JWT |
| `ADDR` | `:8080` | Adresse d'écoute |
| `DB_PATH` | `data/server.db` | Chemin SQLite serveur |

### Endpoints principaux

| Méthode | Route | Auth |
|---------|-------|------|
| `GET` | `/health` | Non |
| `POST` | `/api/register`, `/api/login` | Non |
| `GET/POST` | `/api/v1/movies` | JWT |
| `GET` | `/api/v1/stats` | JWT |
| `GET/POST` | `/api/v1/sync` | JWT |

## Commandes Makefile

| Commande | Description |
|----------|-------------|
| `make build` | Compile CLI et serveur |
| `make test` | Lance tous les tests |
| `make lint` | Analyse statique (golangci-lint) |
| `make run-cli` | Build + lance la TUI |
| `make run-server` | Build + lance l'API |
| `make docker-up` | Démarre le serveur via Docker Compose |
| `make docker-down` | Arrête les conteneurs |
| `make tidy` | `go mod tidy` |

## Tests E2E manuels

Checklist complète : [docs/E2E.md](docs/E2E.md)

## Architecture

```
cmd/cli/              # Binaire TUI
cmd/server/           # Binaire API
internal/config/      # Config YAML (~/.movietracker/)
internal/client/      # Client HTTP auth + sync
internal/sync/        # Service sync hybride LWW
internal/domain/      # Entités métier
internal/database/    # Migrations SQLite (goose)
internal/repository/  # Accès données
internal/service/     # Logique métier
internal/server/      # Handlers HTTP + middleware JWT
internal/tui/         # Interface Bubble Tea
internal/tui/messages/# Chaînes FR centralisées
internal/version/     # Version applicative (1.0.0)
migrations/           # Schémas client et serveur
```

## Licence

Projet pédagogique ESGI — voir [PLAN.md](PLAN.md).
