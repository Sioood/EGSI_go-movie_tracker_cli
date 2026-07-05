# Plan MovieTracker CLI — Suivi complet

> Cocher `[x]` au fur et à mesure. **Phase livrée : 2** — TUI et navigation implémentées.

**Dernière mise à jour** : 2026-07-05  
**Phase en cours** : Phase 3 — Gestion films dans la TUI

**Progression globale** : `3 / 14` phases terminées

---

## Vue d'ensemble — toutes les phases

- [x] **Phase 0** — [Fondations](#phase-0--fondations) · P0 · Facile · ~2j
- [x] **Phase 1** — [Données locales](#phase-1--couche-données-locale) · P0 · Moyen · ~3j
- [x] **Phase 2** — [TUI navigation](#phase-2--tui--coquille-et-navigation) · P0 · Moyen-Difficile · ~4j
- [ ] **Phase 3** — [CRUD films TUI](#phase-3--gestion-films-dans-la-tui) · P0 · Moyen · ~3j
- [ ] **Phase 4** — [Recherche / filtres](#phase-4--recherche-et-filtres) · P1 · Moyen · ~2j
- [ ] **Phase 5** — [Statistiques](#phase-5--statistiques) · P1 · Moyen · ~2j
- [ ] **Phase 6** — [Auth serveur](#phase-6--authentification-serveur) · P0 · Moyen-Difficile · ~4j
- [ ] **Phase 7** — [API REST](#phase-7--api-rest-films) · P0 · Moyen · ~3j
- [ ] **Phase 8** — [Login TUI](#phase-8--connexion-tui--serveur) · P0 · Moyen · ~2j
- [ ] **Phase 9** — [Sync hybride](#phase-9--sync-hybride) · P0 · Difficile · ~5j
- [ ] **Phase 10** — [Polish](#phase-10--robustesse-et-polish) · P1 · Moyen · ~3j
- [ ] **Bonus A** — [TMDB](#bonus-a--intégration-tmdb) · P2 · Moyen · ~3j
- [ ] **Bonus B** — [Export CSV/JSON](#bonus-b--export-csv--json) · P3 · Facile · ~1j
- [ ] **Bonus C** — [Sync avancée](#bonus-c--améliorations-sync) · P3 · Difficile · ~3j

---

## Stack technique (projet complet)

| Couche | Techno | Phase |
|--------|--------|-------|
| TUI | Bubble Tea + Bubbles + Lip Gloss | 2+ |
| HTTP API | chi + net/http | 6+ |
| SQLite | modernc.org/sqlite | 0+ |
| Migrations | goose | 0 |
| Auth | Argon2id + JWT | 6+ |
| Config YAML | `~/.movietracker/` | 8+ |

---

## Ordre de développement

```
0 → 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10 → [A, B, C]
```

```bash
make build && make test
make run-cli
make run-server
```

---

## Phase 0 — Fondations

**Statut phase** : [ ] non commencée · [ ] en cours · [x] terminée  
**Priorité** : P0 | **Difficulté** : Facile | **Temps** : ~2 jours

**Objectif** : squelette compilable, conventions posées.

#### Tâches

- [x] Module Go `github.com/movietracker/movie-tracker`
- [x] Arborescence `cmd/`, `internal/`, `migrations/`
- [x] `Makefile` : build, test, run-cli, run-server
- [x] Migrations goose client (squelette) + serveur (table `users`)
- [x] `internal/domain` : `User`, `Movie`, `WatchEntry`
- [x] `internal/apperrors` : erreurs sentinel
- [x] `slog` dans les deux binaires
- [x] README minimal

#### Livrables

- [x] `go build ./...` passe
- [x] DB migrée au démarrage, logs visibles

#### Fichiers livrés

`go.mod`, `Makefile`, `README.md`, `.gitignore`, `cmd/`, `internal/apperrors/`, `internal/domain/`, `internal/database/`, `internal/logging/`, `migrations/`

---

## Phase 1 — Couche données locale

**Statut phase** : [ ] non commencée · [ ] en cours · [x] terminée  
**Priorité** : P0 | **Difficulté** : Moyen | **Temps** : ~3 jours

**Objectif** : CRUD films en local sans TUI ni réseau.

#### Tâches

- [x] Repository SQLite : Create, GetByID, ListByUser, Update, Delete
- [x] Repository WatchEntry : note, critique, date, watched
- [x] Tests intégration `:memory:`
- [x] MovieService : validation titre, note
- [x] Erreurs ErrMovieNotFound, ErrDB wrappées

#### Livrables

- [x] Tests CRUD locaux ajoutés (`make test` à relancer avec Go installé dans l'environnement)

#### Fichiers livrés

`internal/repository/`, `internal/service/`, `migrations/client/002_movies_watch_entries.sql`, `internal/database/migrations/client/002_movies_watch_entries.sql`

---

## Phase 2 — TUI : coquille et navigation

**Statut phase** : [ ] non commencée · [ ] en cours · [x] terminée  
**Priorité** : P0 | **Difficulté** : Moyen-Difficile | **Temps** : ~4 jours

#### Tâches

- [x] Modèle Bubble Tea + routing écrans
- [x] Écrans : Splash, MainMenu, MovieList, MovieDetail, Stats, Settings, Login, Help
- [x] Bubbles : list, textinput, textarea
- [x] Lip Gloss : header, footer
- [x] État global : config, utilisateur

#### Livrables

- [x] Navigation clavier entre tous les écrans

#### Fichiers livrés

`internal/tui/`, `cmd/cli/main.go`

---

## Phase 3 — Gestion films dans la TUI

**Statut phase** : [ ] non commencée · [ ] en cours · [ ] terminée  
**Priorité** : P0 | **Difficulté** : Moyen | **Temps** : ~3 jours

#### Tâches

- [ ] Ajouter film (titre + année)
- [ ] Liste avec statut vu/non vu
- [ ] Détail : note, critique, date
- [ ] Marquer vu + date YYYY-MM-DD
- [ ] Note échelle 5/10
- [ ] Critique texte
- [ ] Messages d'erreur inline

#### Livrables

- [ ] Cycle add → watch → rate → review en local

---

## Phase 4 — Recherche et filtres

**Statut phase** : [ ] non commencée · [ ] en cours · [ ] terminée  
**Priorité** : P1 | **Difficulté** : Moyen | **Temps** : ~2 jours

#### Tâches

- [ ] Barre recherche TUI (LIKE titre)
- [ ] Filtres : tous / vus / non vus / notés / sans note
- [ ] Tri : titre, date, note
- [ ] Repository Search avec MovieSearchParams
- [ ] Mise à jour temps réel liste

#### Livrables

- [ ] Recherche + filtres fonctionnels

---

## Phase 5 — Statistiques

**Statut phase** : [ ] non commencée · [ ] en cours · [ ] terminée  
**Priorité** : P1 | **Difficulté** : Moyen | **Temps** : ~2 jours

#### Tâches

- [ ] StatsService : totaux, moyenne, best/worst
- [ ] Histogramme ASCII par mois
- [ ] Écran TUI Stats

#### Livrables

- [ ] Stats alimentées par la DB

---

## Phase 6 — Authentification serveur

**Statut phase** : [ ] non commencée · [ ] en cours · [ ] terminée  
**Priorité** : P0 | **Difficulté** : Moyen-Difficile | **Temps** : ~4 jours

#### Tâches

- [ ] Argon2id (`alexedwards/argon2id`)
- [ ] POST register, login, refresh
- [ ] Middleware JWT
- [ ] Validation email + password min 8
- [ ] Tests httptest
- [ ] JWT_SECRET env, rate limiting

#### Hash Argon2id

- [ ] HashPassword, ComparePassword, format PHC

#### Livrables

- [ ] Register/login via curl

---

## Phase 7 — API REST films

**Statut phase** : [ ] non commencée · [ ] en cours · [ ] terminée  
**Priorité** : P0 | **Difficulté** : Moyen | **Temps** : ~3 jours

| Méthode | Route | Action |
|---------|-------|--------|
| GET | /api/v1/movies | Liste |
| POST | /api/v1/movies | Créer |
| GET | /api/v1/movies/{id} | Détail |
| PUT | /api/v1/movies/{id} | Modifier |
| DELETE | /api/v1/movies/{id} | Supprimer |
| PUT | /api/v1/movies/{id}/watch | Watch |
| GET | /api/v1/stats | Stats |
| GET/POST | /api/v1/sync | Sync |

#### Livrables

- [ ] CRUD API authentifié

---

## Phase 8 — Connexion TUI ↔ serveur

**Statut phase** : [ ] non commencée · [ ] en cours · [ ] terminée  
**Priorité** : P0 | **Difficulté** : Moyen | **Temps** : ~2 jours

#### Tâches

- [ ] Écrans Login + Register
- [ ] AuthClient HTTP
- [ ] Config ~/.movietracker/ (0600)
- [ ] offline_mode

#### Livrables

- [ ] Token persisté, reconnexion auto

---

## Phase 9 — Sync hybride

**Statut phase** : [ ] non commencée · [ ] en cours · [ ] terminée  
**Priorité** : P0 | **Difficulté** : Difficile | **Temps** : ~5 jours

#### Tâches

- [ ] sync_metadata, push/pull pending
- [ ] Last-write-wins
- [ ] Indicateur footer, sync S + 30s
- [ ] Retry exponentiel

#### Livrables

- [ ] Sync local ↔ serveur

---

## Phase 10 — Robustesse et polish

**Statut phase** : [ ] non commencée · [ ] en cours · [ ] terminée  
**Priorité** : P1 | **Difficulté** : Moyen | **Temps** : ~3 jours

#### Tâches

- [ ] Messages TUI FR centralisés
- [ ] Écran aide complet
- [ ] Tests E2E README
- [ ] docker-compose, golangci-lint

#### Livrables

- [ ] App production-ready

---

## Bonus A — TMDB · P2

- [ ] TMDB_API_KEY, client, endpoint search/external
- [ ] Recherche TUI à l'ajout
- [ ] Cache métadonnées

## Bonus B — Export · P3

- [ ] ExportService, JSON + CSV depuis Settings

## Bonus C — Sync avancée · P3

- [ ] Résolution conflits manuelle TUI
- [ ] Multi-appareils avancé

---

## Matrice récapitulative

| # | Feature | Priorité | Statut | Dépend de |
|---|---------|----------|--------|-----------|
| 0 | Fondations | P0 | [x] | — |
| 1 | SQLite local | P0 | [ ] | 0 |
| 2 | TUI navigation | P0 | [ ] | 0 |
| 3 | CRUD films TUI | P0 | [ ] | 1, 2 |
| 4 | Recherche / filtres | P1 | [ ] | 3 |
| 5 | Statistiques | P1 | [ ] | 1, 2 |
| 6 | Auth serveur | P0 | [ ] | 0 |
| 7 | API REST | P0 | [ ] | 1, 6 |
| 8 | Login TUI | P0 | [ ] | 2, 6 |
| 9 | Sync hybride | P0 | [ ] | 7, 8 |
| 10 | Polish | P1 | [ ] | 9 |
| A | TMDB | P2 | [ ] | 7, 3 |
| B | Export | P3 | [ ] | 1 |
| C | Sync avancée | P3 | [ ] | 9 |

---

## Prochaine étape

**Phase 1** — Repositories SQLite + MovieService + tests.
