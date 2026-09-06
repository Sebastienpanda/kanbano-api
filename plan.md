# Plan — Kanbano API

Suivi des chantiers du projet : décisions prises, état d'avancement, backlog.
Mis à jour à chaque fin de chapitre pour garder le fil sans dépendre de l'historique de conversation.

---

## Chapitres terminés

### ✅ Logs (Administration > Logs)

**Objectif** : traçabilité (erreurs, latence, qui a fait quoi) + base pour notification Discord à distance.

**Schéma** — table `logs` (migration `000030_create_logs`) :
```sql
CREATE TABLE logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    level       TEXT NOT NULL CHECK (level IN ('info', 'warning', 'error')),
    message     TEXT NOT NULL,
    source      TEXT NOT NULL,
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    request_id  UUID,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- + idx_logs_level_created_at, idx_logs_created_at, idx_logs_user_id, idx_logs_request_id
```
- Pas d'email stocké, uniquement `user_id` (résolu côté affichage) — RGPD.
- `user_id` NULL = système (job interne, websocket...).

**Implémenté** :
- `internal/models/log.go`, `internal/repository/log.repository.go` (`List`, `Insert`)
- `internal/middleware/logging.go` — `NewRequestLogger` : log JSON stdout (slog) par requête, niveau selon status
- `internal/middleware/request_id.go` — génère un vrai UUID par requête (`WithRequestID`/`RequestIDFromContext`), partagé stdout + DB
- `internal/middleware/admin.go` — `AdminRequired`, whitelist via env `ADMIN_USER_IDS` (refuse tout si vide)
- `internal/handler/log.handler.go` + `internal/routes/log.routes.go` — `GET /api/v1/admin/logs?level=&limit=&offset=`
- Écriture auto en base (async, non bloquant, `context.Background()` + timeout 5s) :
  - `error` sur toute erreur serveur (500), via `serverError`/`handleRepoError` (internal/handler/errors.go) — message réel, route, user_id, request_id
  - `warning` sur requête lente (`status < 500` et latence ≥ seuil), via le middleware — seuil configurable `SLOW_REQUEST_THRESHOLD_MS` (défaut 1000ms)

**Config requise (.env)** :
- `ADMIN_USER_IDS` — UUID séparés par virgules, sinon `/admin/logs` refuse tout le monde
- `SLOW_REQUEST_THRESHOLD_MS` — optionnel, défaut 1000

**Non fait / reporté** (voir backlog) :
- Résolution email côté frontend (join/lookup à faire côté front à partir de `user_id`)
- Client webhook Discord applicatif (notif auto warning/error en prod)
- Rate-limit sur les notifs Discord
- Politique de rétention des logs (purge après X jours ?)

---

## Chapitre en cours

_Aucun — prochain chapitre à définir._

---

## Backlog / idées

- **Webhook Discord applicatif** : notif async (goroutine, non bloquant) sur warning/error uniquement, avec rate-limit basique anti-flood
- **Rétention des logs** : décider durée de purge, job de nettoyage
- **Résolution email admin/logs** : endpoint ou join pour hydrater le frontend à partir des `user_id`

---

## Conventions de suivi

- Un chapitre passe en ✅ **seulement** quand le code est mergé/committé et validé par toi.
- Chaque chapitre fermé liste : objectif, ce qui a été fait (fichiers clés), config requise, ce qui a été volontairement reporté.
- Je propose la mise à jour de ce fichier à chaque fin de chapitre — pas d'automatisation silencieuse du `/compact`.
