-- Rôles d'organisation : 'edit' devient 'admin', 'view' devient 'member'.
-- L'owner n'est pas stocké ici : il reste déterminé par organisations.user_id
-- (un seul owner par organisation).

ALTER TABLE organisation_members
    DROP CONSTRAINT organisation_members_role_check;

UPDATE organisation_members
SET role = CASE role WHEN 'edit' THEN 'admin' ELSE 'member' END;

ALTER TABLE organisation_members
    ALTER COLUMN role SET DEFAULT 'member',
    ADD CONSTRAINT organisation_members_role_check CHECK (role IN ('admin', 'member'));

ALTER TABLE organisation_invitations
    DROP CONSTRAINT organisation_invitations_role_check;

UPDATE organisation_invitations
SET role = CASE role WHEN 'edit' THEN 'admin' ELSE 'member' END
WHERE role IS NOT NULL;

ALTER TABLE organisation_invitations
    ADD CONSTRAINT organisation_invitations_role_check CHECK (role IN ('admin', 'member'));
