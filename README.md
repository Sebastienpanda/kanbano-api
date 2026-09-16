# Kanbano API

API backend de Kanbano — gestion de workspaces, colonnes, tâches et organisations en temps réel.

## Stack

- **Go** 1.26 — [Chi](https://github.com/go-chi/chi) (routeur HTTP) — [pgx v5](https://github.com/jackc/pgx) (driver PostgreSQL)
- **NeonDB** (PostgreSQL) via [golang-migrate](https://github.com/golang-migrate/migrate)
- **Neon Auth** (JWT EdDSA) pour l'authentification
- WebSocket pour la diffusion d'événements temps réel
- Stockage objet compatible S3 pour les avatars

## Structure

```
cmd/
  kanbano/     CLI interne (scaffold, migrations, utilitaires)
  routes/      Génère la liste des routes enregistrées
internal/
  handler/     Handlers HTTP (décodage, validation, appel des repositories)
  repository/  Accès aux données (SQL, pgx) — jamais de SQL ailleurs
  models/      Structures de données partagées
  middleware/  Auth JWT, contrôle admin, logging, request ID
  routes/      Déclaration des routes Chi
  server/      Démarrage / arrêt gracieux du serveur HTTP
  db/          Connexion au pool PostgreSQL
  storage/     Client de stockage objet (avatars)
  ws/          Connexions WebSocket et diffusion d'événements
  media/       Génération des dérivés d'avatar (AVIF/WebP/PNG)
  brevo/       Client API Brevo (emails transactionnels)
  utils/       Helpers HTTP partagés (JSON, validation)
  logging/     Logger slog structuré partagé
```

## Prérequis

- Go 1.26+
- [golang-migrate](https://github.com/golang-migrate/migrate) (`migrate` en PATH)
- Un fichier `.env` à la racine avec les variables de connexion (base de données, JWKS Neon Auth, stockage objet, Brevo — voir `.env.example`)

## Démarrage

```bash
go run main.go
```

## Migrations de base de données

```bash
make migrate-up                # applique les migrations
make migrate-down              # rollback complet
make migrate-status            # affiche la version courante
make migrate-create name=xxx   # crée une nouvelle migration
```

## Autres commandes utiles

```bash
make routes                    # liste les routes enregistrées
make scaffold name=toto        # génère handler/repository/model/routes pour une nouvelle ressource
make install-hooks              # active les git hooks du projet (.githooks)
```

## Conventions

Voir [CLAUDE.md](./CLAUDE.md) pour les conventions de code détaillées (gestion du `user_id`, emplacement du SQL, format des commits, etc.).
