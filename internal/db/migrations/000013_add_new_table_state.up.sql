CREATE TABLE IF NOT EXISTS states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR NOT NULL,
    color VARCHAR,
    user_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (user_id, name)
);

CREATE TRIGGER states_updated_at
    BEFORE UPDATE ON public.states
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at();
