CREATE OR REPLACE FUNCTION touch_workspace_updated_at_from_column()
    RETURNS TRIGGER AS $$
BEGIN
    UPDATE workspaces SET updated_at = NOW() WHERE id = NEW.workspace_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION touch_workspace_updated_at_from_task()
    RETURNS TRIGGER AS $$
BEGIN
    UPDATE workspaces SET updated_at = NOW()
    WHERE id = (SELECT workspace_id FROM columns WHERE id = NEW.column_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER columns_touch_workspace_updated_at
    AFTER INSERT OR UPDATE ON public.columns
    FOR EACH ROW
EXECUTE FUNCTION touch_workspace_updated_at_from_column();

CREATE TRIGGER tasks_touch_workspace_updated_at
    AFTER INSERT OR UPDATE ON public.tasks
    FOR EACH ROW
EXECUTE FUNCTION touch_workspace_updated_at_from_task();
