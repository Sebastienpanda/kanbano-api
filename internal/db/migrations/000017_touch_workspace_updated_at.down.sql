DROP TRIGGER IF EXISTS columns_touch_workspace_updated_at ON public.columns;
DROP TRIGGER IF EXISTS tasks_touch_workspace_updated_at ON public.tasks;
DROP FUNCTION IF EXISTS touch_workspace_updated_at_from_column();
DROP FUNCTION IF EXISTS touch_workspace_updated_at_from_task();
