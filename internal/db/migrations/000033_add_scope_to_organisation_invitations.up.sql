-- Une invitation peut, en plus de faire rejoindre l'organisation, accorder
-- un accès à un workspace ou à une colonne précise (invitation envoyée
-- depuis une tâche). workspace_id/column_id/role restent NULL pour une
-- invitation "organisation seule" comme avant.

ALTER TABLE organisation_invitations
    ADD COLUMN workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    ADD COLUMN column_id    UUID REFERENCES columns(id) ON DELETE CASCADE,
    ADD COLUMN role         TEXT CHECK (role IN ('view', 'edit'));

ALTER TABLE organisation_invitations
    ADD CONSTRAINT organisation_invitations_column_requires_workspace
        CHECK (column_id IS NULL OR workspace_id IS NOT NULL);

ALTER TABLE organisation_invitations
    ADD CONSTRAINT organisation_invitations_role_requires_workspace
        CHECK (role IS NULL OR workspace_id IS NOT NULL);
