ALTER TABLE public.workspaces ADD COLUMN description TEXT;
ALTER TABLE workspaces RENAME COLUMN name TO title;
ALTER TABLE workspaces ALTER COLUMN title TYPE VARCHAR(255);
