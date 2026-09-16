ALTER TABLE organisation_invitations
    DROP CONSTRAINT IF EXISTS organisation_invitations_role_requires_workspace,
    DROP CONSTRAINT IF EXISTS organisation_invitations_column_requires_workspace,
    DROP COLUMN IF EXISTS role,
    DROP COLUMN IF EXISTS column_id,
    DROP COLUMN IF EXISTS workspace_id;
