DROP TRIGGER IF EXISTS access_grants_column_workspace_consistency ON access_grants;
DROP FUNCTION IF EXISTS access_grants_check_column_workspace();

ALTER TABLE tags
    ALTER COLUMN name TYPE VARCHAR;

ALTER TABLE organisation_invitations
    DROP CONSTRAINT organisation_invitations_email_format;
