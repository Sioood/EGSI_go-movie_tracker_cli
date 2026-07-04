# MovieTracker CLI

Application terminal (TUI) en Go pour suivre vos films : notes, critiques, statistiques et synchronisation avec un serveur distant.

## Prérequis

- Go 1.22+
- Make (optionnel)

## Installation

```bash
go mod tidy
make build
```

## Lancement

### CLI (TUI)

```bash
make run-cli
# ou
./bin/movietracker
```

La configuration locale est stockée dans `~/.movietracker/config.yaml`.
La base SQLite locale : `~/.movietracker/client.db`.

### Serveur API

```bash
export JWT_SECRET="votre-secret"
make run-server
# ou
./bin/movietracker-server
```

Variables d'environnement serveur :

| Variable | Description | Défaut |
|----------|-------------|--------|
| `JWT_SECRET` | Secret JWT | `dev-secret-change-me` |
| `ADDR` | Adresse d'écoute | `:8080` |
| `DB_PATH` | Fichier SQLite | `data/server.db` |
| `TMDB_API_KEY` | Clé API TMDB (bonus) | — |

## Raccourcis TUI

| Touche | Action |
|--------|--------|
| `↑/↓` | Navigation |
| `Entrée` | Valider |
| `Esc` | Retour menu |
| `q` | Quitter |
| `?` | Aide |
| `S` | Synchroniser |
| `/` | Rechercher (liste films) |
| `Tab` | Changer filtre |
| `Ctrl+N` | Changer tri |

## API REST

### Auth

```bash
# Inscription
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'

# Connexion
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'
```

### Films (authentifié)

```bash
TOKEN="..."
curl http://localhost:8080/api/v1/movies -H "Authorization: Bearer $TOKEN"
```

## Tests

```bash
make test
```

### Tests E2E manuels

1. Lancer le serveur et la CLI
2. Créer un compte via l'écran Connexion
3. Ajouter un film, le marquer vu, noter et critiquer
4. Vérifier les statistiques
5. Appuyer sur `S` pour synchroniser
6. Exporter en JSON/CSV depuis Paramètres

## Docker

```bash
docker compose up --build
```

## Architecture

```
cmd/cli          → TUI Bubble Tea
cmd/server       → API REST (chi)
internal/domain  → Modèles et interfaces
internal/repository → SQLite
internal/service → Logique métier
internal/api     → Handlers HTTP
internal/auth    → Argon2id + JWT
internal/sync    → Moteur de sync hybride
internal/tui     → Interface terminal
```

## Licence

Projet éducatif ESGI.
# EGSI_go-movie_tracker_cli
