ALTER TABLE tasks RENAME COLUMN status TO tag;
ALTER TYPE task_status RENAME TO task_tag;
