CREATE TABLE task_assignees (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id     UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    member_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT NOT NULL DEFAULT 'view' CHECK (role IN ('view', 'edit')),
    assigned_by UUID NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX task_assignees_unique ON task_assignees (task_id, member_id);

ALTER TABLE organisation_invitations
    ADD COLUMN task_id UUID REFERENCES tasks(id) ON DELETE CASCADE;

ALTER TABLE organisation_invitations
    ADD CONSTRAINT organisation_invitations_task_requires_column
        CHECK (task_id IS NULL OR column_id IS NOT NULL);
