REST API de l'application kanban Kanbano (workspaces, colonnes, tâches, tags,
organisations, utilisateurs).

Conventions communes à toutes les routes ci-dessous. Pour chaque route, la
description de chaque code d'erreur liste les valeurs possibles de `error`,
séparées par ` | `.

# Conventions

## URL et authentification

- Toutes les routes métier sont sous `/api/v1` (ex. `GET /api/v1/workspaces`).
- Authentification : header `Authorization: Bearer <JWT Neon Auth>`.
  L'utilisateur courant est déduit du token : aucune route n'attend son ID
  dans le body ou l'URL.
- WebSocket `GET /api/v1/ws` : le token passe dans le header
  `Sec-WebSocket-Protocol: bearer, <token>` (pas dans `Authorization`).
- Rate limit par IP : 10 requêtes/s, rafale de 30.
- Hors `/api/v1` : `GET /health` et `POST /webhooks/brevo` (non destinés au front).

## Corps de requête

- Header `Content-Type: application/json` obligatoire, **valeur exacte**
  (`application/json; charset=utf-8` est refusé) → sinon `415`.
- Les champs inconnus sont refusés → `400 {"error":"could not decode body"}`.
- Un JSON mal formé ou un type incorrect (ex. nombre au lieu de chaîne)
  → `400 {"error":"could not decode body"}`.
- PATCH : seuls les champs envoyés sont modifiés ; un champ absent est ignoré.

## Réponses de succès

| Cas | Statut | Corps |
|---|---|---|
| Lecture | `200` | la ressource ou un tableau (schéma indiqué sur chaque route) |
| Création | `201` | `{"id": "<uuid>", "status": "created"}` |
| Modification | `200` | `{"status": "updated"}` |
| Suppression | `204` | aucun corps |

Après une création ou une modification, le front doit relire la ressource
(ou s'appuyer sur l'événement WebSocket) : la réponse ne la renvoie pas.

## Format des erreurs

### Erreur standard (JSON)

```json
{ "error": "workspace not found", "args": null }
```

- `error` : message en anglais, stable, utilisable comme clé de traduction.
- `args` : paramètres éventuels du message, `null` le plus souvent.

### Erreur de validation du body — `400`

```json
{ "errors": { "name": "<message en français>", "email": "<message en français>" } }
```

- Clé = nom JSON du champ, valeur = message déjà traduit **en français**.
- Pour un tableau d'objets (ex. `members[].role`), la clé est le nom du champ
  seul (`role`), sans l'index.
- Les règles de validation de chaque champ sont dans le schéma du body :
  `required`, `minLength`, `maxLength`, `enum`, `format`…
- Distinguer les deux formats d'erreur `400` par la présence de `errors`
  (validation) ou de `error` (autre erreur) : c'est le schéma
  `BadRequestResponse`.
- Un `422` n'existe que pour un `status` de tâche invalide
  (`{"error":"invalid status"}`).

### Erreurs en texte brut (pas du JSON)

Ces erreurs sortent des middlewares, avant les handlers, en `text/plain` :

| Statut | Corps | Cause |
|---|---|---|
| `401` | `missing token` | header `Authorization` absent ou sans `Bearer ` |
| `401` | `invalid token` | token expiré, mal signé ou invalide |
| `429` | `too many requests` | rate limit dépassé |

Le front ne doit donc pas parser le corps en JSON sur un `401` ou un `429`.

## Sens des codes d'erreur

| Statut | Signification |
|---|---|
| `400` | requête invalide : UUID mal formé dans l'URL (`invalid id`, `invalid columnId`…), `limit`/`offset` invalide, body illisible ou validation échouée |
| `401` | pas authentifié (voir ci-dessus) : rediriger vers la connexion |
| `403` | la ressource est visible, mais le rôle de l'utilisateur ne permet pas cette action (ex. `organisation owner or admin required`) |
| `404` | la ressource n'existe pas, a été supprimée, **ou l'utilisateur n'y a pas accès** |
| `409` | conflit (ex. `an invitation is already pending for this email`) |
| `415` | `Content-Type` absent ou différent de `application/json` |
| `422` | valeur refusée par une règle métier (ex. `invalid status`) |
| `429` | rate limit |
| `500` | erreur serveur : `{"error":"internal server error"}` |

### 404 plutôt que 403

Un workspace (et ce qu'il contient) auquel l'utilisateur n'a pas accès répond
`404`, jamais `403` : l'API ne révèle pas l'existence d'une ressource
invisible. Côté front, un `404` sur une ressource affichée juste avant signifie
le plus souvent qu'elle a été supprimée ou que l'accès a été retiré : retirer
la ressource de l'état local et revenir à la liste.

## Rôles

- Organisation : `owner` (unique, créateur), `admin`, `member`.
- Workspace : accès `view` (lecture seule) ou `edit` (colonnes et tâches).
- Les droits effectifs sur un workspace sont renvoyés sous forme de règles
  CASL par `GET /api/v1/workspaces/{id}/abilities` : le front
  doit s'appuyer sur ces règles plutôt que de recalculer les droits.

## Pagination

Routes paginées : query `limit` (défaut 50, max 200, au-delà ramené à 200) et
`offset` (défaut 0). Une valeur non entière ou négative → `400`.

## Temps réel (WebSocket)

Chaque message reçu a la forme :

```json
{ "type": "task.updated", "workspace_id": "<uuid>", "data": { }, "recent": [ ] }
```

Types : `workspace.created`, `workspace.updated`, `workspace.deleted`,
`column.created`, `column.updated`, `column.deleted`, `task.created`,
`task.updated`, `task.deleted`, `user.updated`, `avatar.updated`,
`avatar.deleted`. `workspace_id`, `data` et `recent` sont omis quand ils ne
s'appliquent pas.

## Types JSON

- Identifiants : UUID en chaîne.
- Dates : chaînes RFC 3339 (ex. `2026-09-25T14:03:00Z`).
- Réponses : tous les champs sont présents (listés dans `required`). Ceux qui
  peuvent valoir `null` portent `x-nullable: true` dans le schéma
  (ex. `status`, `tag`, `updated_at`).
- Bodies : un champ absent de `required` est optionnel.
