-- Règles d'accès décrites une seule fois, lues par toutes les requêtes qui
-- ont besoin de savoir qui voit quoi (listes de workspaces, résolution des
-- rôles, abilities).

-- Accès des membres à un workspace non supprimé : l'owner de l'organisation
-- voit tous ses workspaces ('edit'), un admin ou un member seulement ceux
-- partagés avec lui (workspace_members.visibility = 'public'). La jointure
-- sur organisation_members retire automatiquement l'accès d'un utilisateur
-- qui quitte l'organisation.
CREATE VIEW workspace_access AS
SELECT w.id      AS workspace_id,
       o.user_id AS user_id,
       'owner'   AS org_role,
       'edit'    AS workspace_role
FROM workspaces w
JOIN organisations o ON o.id = w.organisation_id
WHERE w.deleted_at IS NULL
UNION ALL
SELECT w.id,
       wm.member_id,
       om.role,
       wm.role
FROM workspace_members wm
JOIN workspaces w
  ON w.id = wm.workspace_id
 AND w.deleted_at IS NULL
JOIN organisation_members om
  ON om.organisation_id = w.organisation_id
 AND om.member_id = wm.member_id
WHERE wm.visibility = 'public';

-- Accès des guests : une invitation acceptée sur une tâche non supprimée,
-- d'une colonne non supprimée, d'un workspace non supprimé.
CREATE VIEW task_guest_access AS
SELECT tg.task_id,
       t.column_id,
       c.workspace_id,
       tg.user_id,
       tg.role
FROM task_guests tg
JOIN tasks t
  ON t.id = tg.task_id
 AND t.deleted_at IS NULL
JOIN columns c
  ON c.id = t.column_id
 AND c.deleted_at IS NULL
JOIN workspaces w
  ON w.id = c.workspace_id
 AND w.deleted_at IS NULL
WHERE tg.status = 'accepted';
