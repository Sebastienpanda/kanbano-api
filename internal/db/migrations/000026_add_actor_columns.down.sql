ALTER TABLE tasks
    DROP COLUMN created_by,
    DROP COLUMN updated_by,
    DROP COLUMN deleted_by;

ALTER TABLE columns
    DROP COLUMN created_by,
    DROP COLUMN updated_by,
    DROP COLUMN deleted_by;

ALTER TABLE tags
    DROP COLUMN updated_by,
    DROP COLUMN deleted_by;
ALTER TABLE tags RENAME CONSTRAINT tags_created_by_name_key TO tags_user_id_name_key;
ALTER TABLE tags RENAME CONSTRAINT tags_created_by_fkey TO tags_user_id_fkey;
ALTER TABLE tags RENAME COLUMN created_by TO user_id;

ALTER TABLE workspaces
    DROP COLUMN updated_by,
    DROP COLUMN deleted_by;
ALTER TABLE workspaces RENAME CONSTRAINT workspaces_created_by_fkey TO boards_user_id_fkey;
ALTER TABLE workspaces RENAME COLUMN created_by TO user_id;
