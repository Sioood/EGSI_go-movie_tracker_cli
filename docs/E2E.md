# Checklist E2E — MovieTracker v1.0

Tests manuels reproductibles avant release. Cocher chaque scénario après validation.

## Validation automatisée (2026-07-08)

Exécutée localement après l'audit phases 0–10 :

| Vérification | Commande | Résultat |
|--------------|----------|----------|
| Tests unitaires/intégration | `make test` | 79 tests OK (16 packages) |
| Lint | `make lint` | 0 issues |
| Build | `make build` | OK |
| Health serveur | `curl /health` | `{"status":"ok","version":"1.0.0"}` |
| Register API | `POST /api/register` | 201 + tokens |
| Movies sans auth | `GET /api/v1/movies` | 401 |
| JWT typ claim | tests `TestAccessTokenCannotRefresh`, `TestRefreshTokenCannotAccessProtectedRoute` | OK |
| Rate limiting | test `TestRateLimitReturns429` | OK |

Les scénarios TUI interactifs ci-dessous restent **manuels** (nécessitent `make run-cli`).

**Prérequis communs :**

```bash
make build
make test
export JWT_SECRET="test-secret-e2e-min-32-chars-long"
make run-server   # terminal 1
make run-cli      # terminal 2
```

Fichiers utilisateur : `~/.config/movietracker/config.json`, `state.json`, `session.json`.

---

## 1. Mode hors ligne

### 1.1 CRUD film

- **Prérequis** : TUI lancée, mode hors ligne activé (Paramètres → `o`)
- **Étapes** : Menu → Films → `a` → titre « Inception », année 2010 → Entrée
- **Résultat** : Film ajouté, visible dans la liste
- **Statut** : [ ]

### 1.2 Note et critique

- **Prérequis** : Film « Inception » dans la liste
- **Étapes** : Sélectionner → Entrée → note 9 → critique « Excellent » → Entrée
- **Résultat** : « Détail enregistré », statut « vu · note 9.0/10 »
- **Statut** : [ ]

### 1.3 Marquer vu / non vu

- **Prérequis** : Écran détail d'un film
- **Étapes** : `u` puis `w`
- **Résultat** : Date du jour après `w`, champs vidés après `u`
- **Statut** : [ ]

### 1.4 Suppression

- **Prérequis** : Au moins un film en liste
- **Étapes** : Sélectionner → `d`
- **Résultat** : « Film supprimé », disparu de la liste
- **Statut** : [ ]

### 1.5 Recherche et filtres

- **Prérequis** : Plusieurs films (vus et non vus)
- **Étapes** : `/` → taper un titre → `f` (cycle filtres) → `t` (cycle tris) → `c` (reset)
- **Résultat** : Liste filtrée/triée, reset restaure tout
- **Statut** : [ ]

### 1.6 Statistiques

- **Prérequis** : Films notés et vus
- **Étapes** : Menu → Statistiques
- **Résultat** : Totaux, moyenne, histogramme mensuel affichés
- **Statut** : [ ]

---

## 2. Authentification

### 2.1 Inscription

- **Prérequis** : Serveur démarré, hors ligne désactivé
- **Étapes** : Connexion → `r` → email + mot de passe (≥ 8 car.) → Entrée
- **Résultat** : « Compte créé pour … », retour menu, footer « en ligne »
- **Statut** : [ ]

### 2.2 Connexion

- **Prérequis** : Compte existant, déconnecté
- **Étapes** : Connexion → identifiants → Entrée
- **Résultat** : « Connecté en tant que … »
- **Statut** : [ ]

### 2.3 Session restaurée

- **Prérequis** : Connecté une fois, TUI fermée
- **Étapes** : Relancer `make run-cli`
- **Résultat** : Session active sans re-saisie (si token valide)
- **Statut** : [ ]

### 2.4 Déconnexion

- **Prérequis** : Connecté
- **Étapes** : Paramètres → `d`
- **Résultat** : « Déconnecté », header « non connecté »
- **Statut** : [ ]

### 2.5 Erreurs auth en français

- **Prérequis** : Serveur arrêté ou mauvais mot de passe
- **Étapes** : Tenter une connexion
- **Résultat** : Message d'erreur en français (pas de texte anglais brut)
- **Statut** : [ ]

---

## 3. Paramètres

### 3.1 URL serveur

- **Prérequis** : Paramètres ouverts
- **Étapes** : Modifier URL → Entrée
- **Résultat** : « Paramètres enregistrés », persistance après redémarrage
- **Statut** : [ ]

### 3.2 Toggle hors ligne

- **Prérequis** : Paramètres ouverts
- **Étapes** : `o` deux fois
- **Résultat** : Footer sync « hors ligne » / « prêt », état persisté
- **Statut** : [ ]

---

## 4. Synchronisation

### 4.1 Push local → serveur

- **Prérequis** : Connecté, film créé en local
- **Étapes** : `S` (sync manuelle)
- **Résultat** : Footer « sync · à jour », film visible côté API
- **Statut** : [ ]

### 4.2 Pull serveur → client

- **Prérequis** : Film créé via API sur un autre client / curl
- **Étapes** : `S` sur la TUI
- **Résultat** : Film apparaît dans la liste locale
- **Statut** : [ ]

### 4.3 Suppression (tombstone)

- **Prérequis** : Film synchronisé
- **Étapes** : Supprimer en local → `S`
- **Résultat** : Film supprimé aussi côté serveur après sync
- **Statut** : [ ]

### 4.4 Sync automatique 30 s

- **Prérequis** : Connecté, en ligne
- **Étapes** : Ajouter un film, attendre ~30 s
- **Résultat** : Footer passe par « en cours… » puis « à jour »
- **Statut** : [ ]

### 4.5 Erreur réseau

- **Prérequis** : Connecté, arrêter le serveur
- **Étapes** : `S`
- **Résultat** : Footer « sync · erreur : … » avec détail en français
- **Statut** : [ ]

### 4.6 Indicateur pending

- **Prérequis** : Modifications locales hors ligne puis reconnexion
- **Étapes** : Désactiver hors ligne, observer le footer
- **Résultat** : « sync · N en attente » puis « à jour » après sync
- **Statut** : [ ]

---

## 5. Serveur API

### 5.1 Health check

```bash
curl -s http://localhost:8080/health
```

- **Résultat** : `{"status":"ok","version":"1.0.0"}`
- **Statut** : [ ]

### 5.2 Register / Login API

```bash
curl -s -X POST http://localhost:8080/api/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"e2e@test.com","password":"password123"}'
```

- **Résultat** : HTTP 201, tokens dans la réponse
- **Statut** : [ ]

### 5.3 Graceful shutdown

- **Étapes** : `Ctrl+C` sur le serveur
- **Résultat** : Log « server stopped », pas de corruption DB
- **Statut** : [ ]

---

## 6. Docker

### 6.1 Démarrage

```bash
cp .env.example .env
# Éditer JWT_SECRET
make docker-up
curl -s http://localhost:8080/health
```

- **Résultat** : Conteneur healthy, health OK
- **Statut** : [ ]

### 6.2 Persistance volume

- **Étapes** : Créer un compte via API → `docker compose down` → `make docker-up`
- **Résultat** : Compte toujours utilisable (login OK)
- **Statut** : [ ]

---

## 7. Interface

### 7.1 Écran aide complet

- **Étapes** : `?` depuis le menu
- **Résultat** : Sections Navigation, Films, Détail, Compte, Sync ; tous les raccourcis listés
- **Statut** : [ ]

### 7.2 Messages 100 % français

- **Étapes** : Provoquer erreurs (titre vide, note invalide, auth échouée)
- **Résultat** : Aucun message anglais affiché à l'utilisateur
- **Statut** : [ ]

---

## Validation finale

```bash
make lint   # 0 issues
make test   # tous verts
make build
```

- **Statut global v1.0** : [x] validation automatisée OK (2026-07-08) — scénarios TUI manuels à cocher avant release utilisateur
