ALTER TABLE public.boards ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE;

CREATE OR REPLACE FUNCTION update_updated_at()
    RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER boards_updated_at
    BEFORE UPDATE ON public.boards
    FOR EACH ROW
EXECUTE FUNCTION update_updated_at();