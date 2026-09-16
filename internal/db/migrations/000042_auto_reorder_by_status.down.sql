DROP TRIGGER IF EXISTS tasks_resequence_after_insert ON tasks;
DROP TRIGGER IF EXISTS tasks_resequence_after_status_update ON tasks;
DROP FUNCTION IF EXISTS tasks_resequence_column_on_status_change();
DROP FUNCTION IF EXISTS tasks_resequence_column(UUID);
DROP FUNCTION IF EXISTS task_status_rank(task_status);
