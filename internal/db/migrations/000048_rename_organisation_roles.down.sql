ALTER TABLE organisation_invitations
    DROP CONSTRAINT organisation_invitations_role_check;

UPDATE organisation_invitations
SET role = CASE role WHEN 'admin' THEN 'edit' ELSE 'view' END
WHERE role IS NOT NULL;

ALTER TABLE organisation_invitations
    ADD CONSTRAINT organisation_invitations_role_check CHECK (role IN ('view', 'edit'));

ALTER TABLE organisation_members
    DROP CONSTRAINT organisation_members_role_check;

UPDATE organisation_members
SET role = CASE role WHEN 'admin' THEN 'edit' ELSE 'view' END;

ALTER TABLE organisation_members
    ALTER COLUMN role SET DEFAULT 'view',
    ADD CONSTRAINT organisation_members_role_check CHECK (role IN ('view', 'edit'));
