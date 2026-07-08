# Déploiement production — MovieTracker

Guide pour héberger le serveur API sur un **VPS** avec **Coolify** (ou Docker Compose manuel).

## Prérequis

- VPS Linux (Ubuntu/Debian recommandé) avec Docker
- [Coolify](https://coolify.io/) installé sur le VPS (ou accès SSH + Docker Compose)
- Nom de domaine pointant vers le VPS (optionnel mais recommandé pour HTTPS)

## Variables d'environnement

| Variable | Exemple | Description |
|----------|---------|-------------|
| `JWT_SECRET` | *(32+ caractères aléatoires)* | **Obligatoire** — signature JWT |
| `DB_PATH` | `/data/server.db` | Chemin SQLite dans le volume persistant |
| `ADDR` | `:8080` | Adresse d'écoute du conteneur |
| `PORT` | `8080` | Alternative PaaS : utilisé si `ADDR` est vide |

Générer un secret :

```bash
openssl rand -base64 32
```

## Option A — Coolify (recommandé)

1. Créer une **nouvelle ressource** → **Docker Compose** ou **Dockerfile**
2. Connecter le dépôt Git du projet
3. Définir les variables d'environnement ci-dessus dans l'interface Coolify
4. Monter un **volume persistant** sur `/data` (pour `DB_PATH=/data/server.db`)
5. Exposer le port `8080` (ou laisser Coolify gérer le proxy HTTPS)
6. Vérifier le healthcheck : `GET /health` → `{"status":"ok","version":"..."}`

### Dockerfile (déjà fourni)

Coolify peut builder directement depuis le [`Dockerfile`](Dockerfile) à la racine :

- Image multi-stage Alpine
- Binaire `movietracker-server` CGO=0
- Utilisateur non-root `appuser`
- Port exposé : `8080`

### Docker Compose

Utiliser [`docker-compose.yml`](docker-compose.yml) ou [`docker-compose.prod.yml`](docker-compose.prod.yml) :

```bash
cp .env.example .env
# Éditer JWT_SECRET
docker compose -f docker-compose.prod.yml up --build -d
curl http://localhost:8080/health
```

## Option B — VPS manuel (sans Coolify)

```bash
git clone <votre-repo>
cd project
cp .env.example .env
# Éditer JWT_SECRET
docker compose up --build -d
```

## Configurer la CLI en production

1. Lancer `movietracker` localement
2. **Paramètres** → URL serveur : `https://api.votre-domaine.com`
3. Désactiver le mode hors ligne (`o`)
4. **Connexion** → créer un compte ou se connecter
5. `S` pour synchroniser les films ; `e`/`i` pour backup config/état

## Backup de la base SQLite

Le volume Docker `server-data` contient `server.db`. Sauvegarder régulièrement :

```bash
docker compose exec server cp /data/server.db /data/server.db.bak.$(date +%Y%m%d)
# ou copier le volume depuis l'hôte
```

## Cross-compilation serveur Linux

```bash
make build-linux
# ou
docker build --platform linux/amd64 -t movietracker-server .
```

## Dépannage

| Problème | Solution |
|----------|----------|
| `JWT_SECRET environment variable is required` | Définir `JWT_SECRET` dans Coolify / `.env` |
| Données perdues au redémarrage | Vérifier le volume persistant sur `/data` |
| CLI ne se connecte pas | URL HTTPS, pare-feu port 443/8080, CORS non requis (client CLI) |
| Healthcheck échoue | Attendre `start_period`, vérifier logs conteneur |
