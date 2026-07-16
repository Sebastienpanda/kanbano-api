CREATE TABLE IF NOT EXISTS columns (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title        VARCHAR(255) NOT NULL,
    position     INT NOT NULL DEFAULT 0,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at   TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TRIGGER columns_updated_at
    BEFORE UPDATE ON public.columns
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at();
