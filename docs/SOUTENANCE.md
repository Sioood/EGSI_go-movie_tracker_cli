# Fiche soutenance — MovieTracker

## 1. Architecture

- **`cmd/cli`** et **`cmd/server`** : deux binaires, point d'entrée
- **`internal/`** : code privé non importable de l'extérieur
- Couches : `domain` → `repository` → `service` → `server` (HTTP) / `tui` (Bubble Tea)
- Pas de dépendance circulaire : le client HTTP vit dans `internal/client`

## 2. TUI Bubble Tea

- Modèle **Elm** : `Init()`, `Update(msg)`, `View()`
- Routing par `Route` (`internal/tui/routes.go`)
- Messages async via `tea.Cmd` (auth, sync, backup)
- Composants **Bubbles** (list, textinput) + styles **Lip Gloss** (thèmes dynamiques)

## 3. Authentification

- Mots de passe : **Argon2id** (format PHC)
- JWT **HS256** : access 15 min, refresh 7 jours, claim `typ`
- Middleware Bearer + rate limiting (5 req/s, burst 20)
- Session persistée dans `~/.config/movietracker/session.json`

## 4. Sync hybride

- Mode hors ligne : SQLite locale (`data/client.db`)
- Mode connecté : push pending → `POST /api/v1/sync`, puis pull → `GET /api/v1/sync`
- Résolution conflits : **last-write-wins** sur `updated_at`
- Sync auto toutes les 30 s + manuelle (`S`)

## 5. Stockage

| Donnée | Emplacement |
|--------|-------------|
| Préférences | `~/.config/movietracker/config.json` |
| État UI | `~/.config/movietracker/state.json` |
| Session | `~/.config/movietracker/session.json` |
| Films local | `data/client.db` |
| Backup serveur | table `user_backups` (JSON par `user_id`) |

## 6. Sécurité

- Fichiers config : permissions **0600**, dossier **0700**
- Isolation multi-tenant : `user_id` sur chaque film + backup
- Pas de secrets dans les logs (`slog` structuré)
- Corps HTTP limité à 1 MiB

## 7. Build et releases

- SQLite pure Go (`modernc.org/sqlite`) → **CGO=0**, cross-compile facile
- **GoReleaser** : CLI Windows/Linux/macOS + serveur Linux
- GitHub Actions : CI (`make test`, `make lint`) + release sur tag `v*`

## 8. Déploiement

- **Docker** multi-stage Alpine, utilisateur non-root
- **Coolify** ou VPS : volume persistant `/data` pour SQLite
- Healthcheck `GET /health`
- Variables : `JWT_SECRET`, `DB_PATH`, `ADDR`/`PORT`

---

## Scénario démo (5 min)

1. **Hors ligne** : lancer CLI → ajouter un film → noter / marquer vu
2. **Paramètres** : montrer `config.json` et `state.json` dans `~/.config/movietracker/`
3. **Thème** : `←`/`→` pour changer les couleurs (midnight / solar / forest)
4. **Connexion** : URL serveur prod → login → sync `S`
5. **Backup** : `e` export config+état serveur, `i` import, `E` export local JSON
6. **API** (optionnel) : `curl /health`, `curl /api/v1/backup` avec Bearer token

---

## Questions fréquentes

**Pourquoi JSON et pas YAML pour la config ?**  
Critère grille ESGI + standard XDG (`~/.config/`).

**Pourquoi deux sync (`/sync` films vs `/backup` config) ?**  
Séparation données métier (films) et préférences/état CLI.

**Pourquoi SQLite et pas PostgreSQL ?**  
Simplicité, embarqué, suffisant pour le projet ; JSONB PostgreSQL cité comme option.

**Comment gérer plusieurs appareils ?**  
Sync films via `/api/v1/sync` ; config/état via `/api/v1/backup`.
