ALTER TABLE tasks ADD COLUMN position_updated_at TIMESTAMP WITH TIME ZONE;

CREATE OR REPLACE FUNCTION set_task_position_updated_at()
    RETURNS TRIGGER AS $$
BEGIN
    IF NEW.position IS DISTINCT FROM OLD.position THEN
        NEW.position_updated_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tasks_position_updated_at
    BEFORE UPDATE ON public.tasks
    FOR EACH ROW
EXECUTE FUNCTION set_task_position_updated_at();
