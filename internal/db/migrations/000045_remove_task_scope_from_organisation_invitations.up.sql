-- Les invitations scopées à une tâche vivent désormais dans task_guests
-- (mécanisme dédié aux invités externes, sans lien avec organisation_members).
-- organisation_invitations redevient exclusivement une invitation à
-- rejoindre l'organisation.

ALTER TABLE organisation_invitations
    DROP CONSTRAINT IF EXISTS organisation_invitations_task_requires_column,
    DROP CONSTRAINT IF EXISTS organisation_invitations_column_requires_workspace;

ALTER TABLE organisation_invitations
    DROP COLUMN IF EXISTS task_id,
    DROP COLUMN IF EXISTS column_id,
    DROP COLUMN IF EXISTS workspace_id;
