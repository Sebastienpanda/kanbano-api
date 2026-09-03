-- Traçabilité des acteurs sur workspaces / columns / tags / tasks.
-- user_id (propriétaire) devient created_by. updated_by / deleted_by restent
-- NULL par défaut et sont renseignés par l'application (repositories) lors
-- d'une modification / suppression.

-- ── workspaces ───────────────────────────────────────────────
ALTER TABLE workspaces RENAME COLUMN user_id TO created_by;
ALTER TABLE workspaces RENAME CONSTRAINT boards_user_id_fkey TO workspaces_created_by_fkey;
ALTER TABLE workspaces
    ADD COLUMN updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN deleted_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- ── tags ─────────────────────────────────────────────────────
ALTER TABLE tags RENAME COLUMN user_id TO created_by;
ALTER TABLE tags RENAME CONSTRAINT tags_user_id_fkey TO tags_created_by_fkey;
ALTER TABLE tags RENAME CONSTRAINT tags_user_id_name_key TO tags_created_by_name_key;
ALTER TABLE tags
    ADD COLUMN updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN deleted_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- ── columns ──────────────────────────────────────────────────
ALTER TABLE columns
    ADD COLUMN created_by UUID REFERENCES users(id) ON DELETE CASCADE,
    ADD COLUMN updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN deleted_by UUID REFERENCES users(id) ON DELETE SET NULL;

UPDATE columns c
SET created_by = w.created_by
FROM workspaces w
WHERE w.id = c.workspace_id;

ALTER TABLE columns ALTER COLUMN created_by SET NOT NULL;

-- ── tasks ────────────────────────────────────────────────────
ALTER TABLE tasks
    ADD COLUMN created_by UUID REFERENCES users(id) ON DELETE CASCADE,
    ADD COLUMN updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN deleted_by UUID REFERENCES users(id) ON DELETE SET NULL;

UPDATE tasks t
SET created_by = w.created_by
FROM columns c
JOIN workspaces w ON w.id = c.workspace_id
WHERE c.id = t.column_id;

ALTER TABLE tasks ALTER COLUMN created_by SET NOT NULL;
