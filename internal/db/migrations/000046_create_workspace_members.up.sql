-- Override du rôle organisation au niveau workspace. Si une ligne existe
-- pour (workspace_id, member_id), son rôle est prioritaire sur le rôle
-- organisation ; sinon le rôle organisation s'applique (même logique que
-- task_assignees vis-à-vis du rôle workspace).
CREATE TABLE IF NOT EXISTS workspace_members (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    member_id    UUID NOT NULL REFERENCES users(id),
    role         TEXT NOT NULL CHECK (role IN ('view', 'edit')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX workspace_members_unique ON workspace_members (workspace_id, member_id);
CREATE INDEX workspace_members_member_id_idx ON workspace_members (member_id);
