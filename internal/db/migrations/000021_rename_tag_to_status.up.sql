ALTER TYPE task_tag RENAME TO task_status;
ALTER TABLE tasks RENAME COLUMN tag TO status;
