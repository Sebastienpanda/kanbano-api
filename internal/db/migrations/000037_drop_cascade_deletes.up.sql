-- Aucune contrainte ON DELETE CASCADE n'est autorisée dans ce schéma : une
-- suppression ne doit jamais se propager silencieusement à des données liées
-- (workspaces/colonnes/tâches d'une organisation entière, par exemple). Le
-- comportement par défaut de Postgres (NO ACTION) bloque la suppression tant
-- que des lignes dépendantes existent, ce qui force une décision explicite
-- côté application plutôt qu'une perte de données en cascade.

ALTER TABLE workspaces DROP CONSTRAINT workspaces_created_by_fkey;
ALTER TABLE workspaces ADD CONSTRAINT workspaces_created_by_fkey
    FOREIGN KEY (created_by) REFERENCES users(id);

ALTER TABLE workspaces DROP CONSTRAINT workspaces_organisation_id_fkey;
ALTER TABLE workspaces ADD CONSTRAINT workspaces_organisation_id_fkey
    FOREIGN KEY (organisation_id) REFERENCES organisations(id);

ALTER TABLE columns DROP CONSTRAINT columns_workspace_id_fkey;
ALTER TABLE columns ADD CONSTRAINT columns_workspace_id_fkey
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id);

ALTER TABLE columns DROP CONSTRAINT columns_created_by_fkey;
ALTER TABLE columns ADD CONSTRAINT columns_created_by_fkey
    FOREIGN KEY (created_by) REFERENCES users(id);

ALTER TABLE tasks DROP CONSTRAINT tasks_column_id_fkey;
ALTER TABLE tasks ADD CONSTRAINT tasks_column_id_fkey
    FOREIGN KEY (column_id) REFERENCES columns(id);

ALTER TABLE tasks DROP CONSTRAINT tasks_created_by_fkey;
ALTER TABLE tasks ADD CONSTRAINT tasks_created_by_fkey
    FOREIGN KEY (created_by) REFERENCES users(id);

ALTER TABLE organisations DROP CONSTRAINT organisations_user_id_fkey;
ALTER TABLE organisations ADD CONSTRAINT organisations_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE organisation_members DROP CONSTRAINT organisation_members_member_id_fkey;
ALTER TABLE organisation_members ADD CONSTRAINT organisation_members_member_id_fkey
    FOREIGN KEY (member_id) REFERENCES users(id);

ALTER TABLE organisation_members DROP CONSTRAINT organisation_members_organisation_id_fkey;
ALTER TABLE organisation_members ADD CONSTRAINT organisation_members_organisation_id_fkey
    FOREIGN KEY (organisation_id) REFERENCES organisations(id);

ALTER TABLE organisation_invitations DROP CONSTRAINT organisation_invitations_organisation_id_fkey;
ALTER TABLE organisation_invitations ADD CONSTRAINT organisation_invitations_organisation_id_fkey
    FOREIGN KEY (organisation_id) REFERENCES organisations(id);

ALTER TABLE organisation_invitations DROP CONSTRAINT organisation_invitations_invited_by_fkey;
ALTER TABLE organisation_invitations ADD CONSTRAINT organisation_invitations_invited_by_fkey
    FOREIGN KEY (invited_by) REFERENCES users(id);

ALTER TABLE organisation_invitations DROP CONSTRAINT organisation_invitations_workspace_id_fkey;
ALTER TABLE organisation_invitations ADD CONSTRAINT organisation_invitations_workspace_id_fkey
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id);

ALTER TABLE organisation_invitations DROP CONSTRAINT organisation_invitations_column_id_fkey;
ALTER TABLE organisation_invitations ADD CONSTRAINT organisation_invitations_column_id_fkey
    FOREIGN KEY (column_id) REFERENCES columns(id);

ALTER TABLE organisation_invitations DROP CONSTRAINT organisation_invitations_task_id_fkey;
ALTER TABLE organisation_invitations ADD CONSTRAINT organisation_invitations_task_id_fkey
    FOREIGN KEY (task_id) REFERENCES tasks(id);

ALTER TABLE access_grants DROP CONSTRAINT access_grants_workspace_id_fkey;
ALTER TABLE access_grants ADD CONSTRAINT access_grants_workspace_id_fkey
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id);

ALTER TABLE access_grants DROP CONSTRAINT access_grants_column_id_fkey;
ALTER TABLE access_grants ADD CONSTRAINT access_grants_column_id_fkey
    FOREIGN KEY (column_id) REFERENCES columns(id);

ALTER TABLE access_grants DROP CONSTRAINT access_grants_member_id_fkey;
ALTER TABLE access_grants ADD CONSTRAINT access_grants_member_id_fkey
    FOREIGN KEY (member_id) REFERENCES users(id);

ALTER TABLE access_grants DROP CONSTRAINT access_grants_granted_by_fkey;
ALTER TABLE access_grants ADD CONSTRAINT access_grants_granted_by_fkey
    FOREIGN KEY (granted_by) REFERENCES users(id);

ALTER TABLE task_assignees DROP CONSTRAINT task_assignees_task_id_fkey;
ALTER TABLE task_assignees ADD CONSTRAINT task_assignees_task_id_fkey
    FOREIGN KEY (task_id) REFERENCES tasks(id);

ALTER TABLE task_assignees DROP CONSTRAINT task_assignees_member_id_fkey;
ALTER TABLE task_assignees ADD CONSTRAINT task_assignees_member_id_fkey
    FOREIGN KEY (member_id) REFERENCES users(id);
