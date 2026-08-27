ALTER TABLE users
    DROP COLUMN IF EXISTS avatar_version,
    DROP COLUMN IF EXISTS avatar_updated_at;
