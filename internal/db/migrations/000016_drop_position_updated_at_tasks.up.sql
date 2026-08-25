DROP TRIGGER IF EXISTS tasks_position_updated_at ON public.tasks;
DROP FUNCTION IF EXISTS set_task_position_updated_at();
ALTER TABLE tasks DROP COLUMN IF EXISTS position_updated_at;
