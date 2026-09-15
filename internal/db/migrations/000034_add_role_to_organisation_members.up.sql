ALTER TABLE organisation_members
    ADD COLUMN role TEXT NOT NULL DEFAULT 'view' CHECK (role IN ('view', 'edit'));

ALTER TABLE organisation_invitations
    DROP CONSTRAINT organisation_invitations_role_requires_workspace;
