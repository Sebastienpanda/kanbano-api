-- updated_at ne doit refléter qu'une vraie modification : NULL tant que la ligne
-- n'a jamais été modifiée, valorisé par le trigger update_updated_at() ensuite.

ALTER TABLE public.columns ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE public.tasks   ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE public.tags    ALTER COLUMN updated_at DROP DEFAULT;

-- Lignes jamais modifiées (updated_at posé par l'ancien DEFAULT NOW()) : on repasse à NULL.
UPDATE public.columns SET updated_at = NULL WHERE updated_at = created_at;
UPDATE public.tasks   SET updated_at = NULL WHERE updated_at = created_at;
UPDATE public.tags    SET updated_at = NULL WHERE updated_at = created_at;
