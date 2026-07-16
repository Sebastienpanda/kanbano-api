DROP TRIGGER IF EXISTS on_auth_user_created ON neon_auth.user;
DROP TRIGGER IF EXISTS on_auth_user_updated ON neon_auth.user;
DROP FUNCTION IF EXISTS public.sync_user_from_auth();