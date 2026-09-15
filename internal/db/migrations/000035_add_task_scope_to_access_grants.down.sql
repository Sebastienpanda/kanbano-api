ALTER TABLE organisation_invitations
    DROP CONSTRAINT organisation_invitations_task_requires_column;

ALTER TABLE organisation_invitations
    DROP COLUMN task_id;

DROP TABLE task_assignees;
