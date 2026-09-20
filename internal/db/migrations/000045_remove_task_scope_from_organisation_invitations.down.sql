ALTER TABLE organisation_invitations
    ADD COLUMN workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    ADD COLUMN column_id    UUID REFERENCES columns(id) ON DELETE CASCADE,
    ADD COLUMN task_id      UUID REFERENCES tasks(id) ON DELETE CASCADE;

ALTER TABLE organisation_invitations
    ADD CONSTRAINT organisation_invitations_column_requires_workspace
        CHECK (column_id IS NULL OR workspace_id IS NOT NULL);

ALTER TABLE organisation_invitations
    ADD CONSTRAINT organisation_invitations_task_requires_column
        CHECK (task_id IS NULL OR column_id IS NOT NULL);
