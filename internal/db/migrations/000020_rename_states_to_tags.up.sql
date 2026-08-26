ALTER TABLE tasks RENAME COLUMN state_id TO tag_id;
ALTER TABLE tasks RENAME CONSTRAINT tasks_state_id_fkey TO tasks_tag_id_fkey;

ALTER TABLE states RENAME TO tags;
ALTER TABLE tags RENAME CONSTRAINT states_pkey TO tags_pkey;
ALTER TABLE tags RENAME CONSTRAINT states_user_id_fkey TO tags_user_id_fkey;
ALTER TABLE tags RENAME CONSTRAINT states_user_id_name_key TO tags_user_id_name_key;
ALTER TRIGGER states_updated_at ON tags RENAME TO tags_updated_at;
