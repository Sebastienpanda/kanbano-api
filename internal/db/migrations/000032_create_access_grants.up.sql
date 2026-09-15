-- Accès accordés à un membre d'organisation sur un workspace, avec une
-- portée soit colonne (column_id renseigné), soit workspace entier
-- (column_id NULL). Le rôle 'view' donne accès en lecture à tout le
-- workspace ; 'edit' autorise la création/modification, bornée à la
-- colonne si column_id est renseigné, ou à tout le workspace sinon.

CREATE TABLE IF NOT EXISTS access_grants (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    column_id    UUID REFERENCES columns(id) ON DELETE CASCADE,
    member_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role         TEXT NOT NULL DEFAULT 'view' CHECK (role IN ('view', 'edit')),
    granted_by   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Un seul grant par (member, workspace) quand la portée est le workspace
-- entier (column_id NULL) ; UNIQUE ignore les NULL en Postgres, d'où
-- l'index partiel.
CREATE UNIQUE INDEX access_grants_workspace_scope_unique
    ON access_grants (member_id, workspace_id)
    WHERE column_id IS NULL;

-- Un seul grant par (member, column) quand la portée est une colonne.
CREATE UNIQUE INDEX access_grants_column_scope_unique
    ON access_grants (member_id, column_id)
    WHERE column_id IS NOT NULL;

CREATE INDEX access_grants_member_idx ON access_grants (member_id);
CREATE INDEX access_grants_workspace_idx ON access_grants (workspace_id);
