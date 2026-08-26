ALTER TRIGGER tags_updated_at ON tags RENAME TO states_updated_at;
ALTER TABLE tags RENAME CONSTRAINT tags_user_id_name_key TO states_user_id_name_key;
ALTER TABLE tags RENAME CONSTRAINT tags_user_id_fkey TO states_user_id_fkey;
ALTER TABLE tags RENAME CONSTRAINT tags_pkey TO states_pkey;
ALTER TABLE tags RENAME TO states;

ALTER TABLE tasks RENAME CONSTRAINT tasks_tag_id_fkey TO tasks_state_id_fkey;
ALTER TABLE tasks RENAME COLUMN tag_id TO state_id;
