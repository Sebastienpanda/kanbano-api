-- Correctifs identifiés lors de la revue PostgreSQL des migrations 000031-000038 :
-- 1) format d'email validé au niveau base (les invitations en TEXT libre
--    pouvaient contenir des valeurs non-email) ;
-- 2) longueur de tags.name alignée sur workspaces/columns/tasks (000038) ;
-- 3) cohérence workspace/column sur access_grants : un grant dont column_id
--    est renseigné doit référencer une colonne appartenant au même
--    workspace_id que le grant (aucune contrainte SQL ne le garantissait).

ALTER TABLE organisation_invitations
    ADD CONSTRAINT organisation_invitations_email_format
        CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$');

ALTER TABLE tags
    ALTER COLUMN name TYPE VARCHAR(50);

CREATE OR REPLACE FUNCTION access_grants_check_column_workspace()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.column_id IS NOT NULL THEN
        IF NOT EXISTS (
            SELECT 1 FROM columns
            WHERE id = NEW.column_id
              AND workspace_id = NEW.workspace_id
        ) THEN
            RAISE EXCEPTION 'access_grants.column_id must belong to access_grants.workspace_id';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER access_grants_column_workspace_consistency
    BEFORE INSERT OR UPDATE ON access_grants
    FOR EACH ROW
    EXECUTE FUNCTION access_grants_check_column_workspace();
