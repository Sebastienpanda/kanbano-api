ALTER TABLE task_assignees
    ADD COLUMN role TEXT NOT NULL DEFAULT 'view' CHECK (role IN ('view', 'edit'));
