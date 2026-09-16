CREATE INDEX workspaces_created_by_idx ON workspaces (created_by);
CREATE INDEX workspaces_organisation_id_idx ON workspaces (organisation_id);

CREATE INDEX columns_workspace_id_idx ON columns (workspace_id) WHERE deleted_at IS NULL;
CREATE INDEX tasks_column_id_idx ON tasks (column_id) WHERE deleted_at IS NULL;

CREATE INDEX organisation_members_organisation_id_idx ON organisation_members (organisation_id);
CREATE INDEX organisation_members_member_id_idx ON organisation_members (member_id);

CREATE INDEX organisation_invitations_workspace_id_idx ON organisation_invitations (workspace_id);
CREATE INDEX organisation_invitations_task_id_idx ON organisation_invitations (task_id);
