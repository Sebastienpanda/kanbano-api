UPDATE public.columns SET updated_at = created_at WHERE updated_at IS NULL;
UPDATE public.tasks   SET updated_at = created_at WHERE updated_at IS NULL;
UPDATE public.tags    SET updated_at = created_at WHERE updated_at IS NULL;

ALTER TABLE public.columns ALTER COLUMN updated_at SET DEFAULT NOW();
ALTER TABLE public.tasks   ALTER COLUMN updated_at SET DEFAULT NOW();
ALTER TABLE public.tags    ALTER COLUMN updated_at SET DEFAULT NOW();
