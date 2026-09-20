CREATE TABLE IF NOT EXISTS access_grants (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    column_id    UUID REFERENCES columns(id) ON DELETE CASCADE,
    member_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role         TEXT NOT NULL DEFAULT 'view' CHECK (role IN ('view', 'edit')),
    granted_by   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX access_grants_workspace_scope_unique
    ON access_grants (member_id, workspace_id)
    WHERE column_id IS NULL;

CREATE UNIQUE INDEX access_grants_column_scope_unique
    ON access_grants (member_id, column_id)
    WHERE column_id IS NOT NULL;

CREATE INDEX access_grants_member_idx ON access_grants (member_id);
CREATE INDEX access_grants_workspace_idx ON access_grants (workspace_id);

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
