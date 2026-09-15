ALTER TABLE organisation_invitations
    ADD CONSTRAINT organisation_invitations_role_requires_workspace
        CHECK (role IS NULL OR workspace_id IS NOT NULL);

ALTER TABLE organisation_members
    DROP COLUMN role;
