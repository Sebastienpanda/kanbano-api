ALTER TABLE workspace_members
    ALTER COLUMN role DROP DEFAULT;

ALTER TABLE workspace_members
    DROP COLUMN visibility;
