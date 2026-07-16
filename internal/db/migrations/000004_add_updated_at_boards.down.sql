DROP TRIGGER IF EXISTS boards_updated_at ON public.boards;
DROP FUNCTION IF EXISTS update_updated_at();
ALTER TABLE public.boards DROP COLUMN updated_at;