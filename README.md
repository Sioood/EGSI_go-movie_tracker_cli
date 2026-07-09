# MovieTracker v1.0

Application terminal en Go pour suivre ses films, avec synchronisation hybride local ↔ serveur. Voir [PLAN.md](PLAN.md) pour la feuille de route complète.

## Prérequis

- Go **1.26+**
- [golangci-lint](https://golangci-lint.run/) (pour `make lint`)
- Docker + Docker Compose (optionnel, pour le serveur API)

## Installation rapide

### Depuis les sources

```bash
make build
make test
```

### Depuis GitHub Releases

Télécharger le binaire pour votre OS depuis [GitHub Releases](https://github.com/Sioood/EGSI_go-movie_tracker_cli/releases) :

| OS | Binaire |
|----|---------|
| Linux amd64 | `movietracker_linux_amd64.tar.gz` |
| macOS amd64/arm64 | `movietracker_darwin_*` |
| Windows amd64 | `movietracker_windows_amd64.zip` |

Le serveur Linux est inclus : `movietracker-server` dans l'archive Linux.

Binaires produits :

- `bin/movietracker` — TUI locale (Bubble Tea)
- `bin/movietracker-server` — API REST

## Utilisation CLI

```bash
make run-cli
```

Au premier lancement :

- La base SQLite locale est créée dans `data/client.db` (répertoire courant)
- La configuration utilisateur est dans `~/.config/movietracker/config.json`
- L'état UI est dans `~/.config/movietracker/state.json`
- La session serveur (tokens) est dans `~/.config/movietracker/session.json`

> Migration automatique depuis l'ancien `~/.movietracker/*.yaml` au premier lancement.

### Modes

| Mode | Description |
|------|-------------|
| **Hors ligne** | CRUD films, notes, critiques et stats en local uniquement |
| **Connecté** | Auth serveur + sync automatique (toutes les 30 s) et manuelle (`S`) |

Depuis le menu : **Films**, **Statistiques**, **Paramètres**, **Connexion**, **Aide** (`?`).

Raccourcis principaux : `/` recherche, `f`/`t`/`c` filtres, `a` ajouter, `ctrl+t` recherche TMDB, `S` synchroniser, `K` conflits, `q` quitter.

### Fonctionnalités bonus

| Bonus | Raccourcis | Description |
|-------|------------|-------------|
| **Export films** | `j` / `J` dans Paramètres | Export JSON / CSV vers `~/.config/movietracker/exports/` |
| **TMDB** | `ctrl+t` dans le formulaire d'ajout | Recherche via proxy serveur `GET /api/v1/search/external` (ou `TMDB_API_KEY` locale) |
| **Sync avancée** | `K` | Résolution manuelle des conflits multi-appareils |

Chaque appareil reçoit un `device_id` persistant dans `config.json`. Les conflits apparaissent dans le footer (`sync · N conflit(s)`).

### Scénarios E2E bonus

1. **Export** : Paramètres → `j` → vérifier `~/.config/movietracker/exports/movies-*.json` ; `J` pour le CSV.
2. **TMDB** : Ajouter un film → `ctrl+t` → chercher un titre → sélectionner → vérifier la référence externe `tmdb:…` dans le détail.
3. **Conflits** : deux instances CLI avec des `device_id` différents → modifier le même film hors ligne → sync → `K` → choisir local/distant → re-sync.

## Serveur API

### Local

```bash
export JWT_SECRET="votre-secret-long-et-aleatoire"
export TMDB_API_KEY="votre-token-tmdb"   # optionnel, active GET /api/v1/search/external
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
| `PORT` | — | Utilisé si `ADDR` vide (compatibilité PaaS) |
| `DB_PATH` | `data/server.db` | Chemin SQLite serveur |

### Endpoints principaux

| Méthode | Route | Auth |
|---------|-------|------|
| `GET` | `/health` | Non |
| `POST` | `/api/register`, `/api/login` | Non |
| `GET/POST` | `/api/v1/movies` | JWT |
| `GET` | `/api/v1/stats` | JWT |
| `GET/POST` | `/api/v1/sync` | JWT |
| `GET/PUT` | `/api/v1/backup/config` | JWT |
| `GET/PUT` | `/api/v1/backup/state` | JWT |
| `GET/PUT` | `/api/v1/backup` | JWT |

## Déploiement production

Guide Coolify / VPS : [docs/DEPLOY.md](docs/DEPLOY.md)

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
| `make build-linux` | Cross-compile Linux (CLI + serveur) |
| `make build-windows` | Cross-compile Windows CLI |
| `make build-darwin` | Cross-compile macOS CLI |
| `make release-snapshot` | Build local GoReleaser (snapshot) |
| `make tidy` | `go mod tidy` |

## Tests E2E manuels

Checklist complète : [docs/E2E.md](docs/E2E.md)  
Fiche soutenance : [docs/SOUTENANCE.md](docs/SOUTENANCE.md)

## Architecture

```
cmd/cli/              # Binaire TUI
cmd/server/           # Binaire API
internal/config/      # Config JSON XDG (~/.config/movietracker/)
internal/client/      # Client HTTP auth + sync + backup + TMDB proxy
internal/sync/        # Service sync hybride LWW + conflits
internal/tmdb/        # Client TMDB
internal/domain/      # Entités métier
internal/database/    # Migrations SQLite (goose)
internal/repository/  # Accès données
internal/service/     # Logique métier
internal/server/      # Handlers HTTP + middleware JWT
internal/tui/         # Interface Bubble Tea
internal/tui/messages/# Chaînes FR centralisées
internal/version/     # Version applicative (1.0.0)
```

Migrations SQL embarquées dans `internal/database/migrations/` (goose embed).

## Licence

Projet pédagogique ESGI — voir [PLAN.md](PLAN.md).
