DROP INDEX IF EXISTS task_guests_accepted_unique;

ALTER TABLE task_guests
    DROP COLUMN expires_at;
