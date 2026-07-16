CREATE OR REPLACE FUNCTION public.sync_user_from_auth()
    RETURNS TRIGGER
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = public
AS $$
BEGIN
    INSERT INTO public.users (id, email, name, created_at)
    VALUES (NEW.id, NEW.email, NULLIF(NEW.name, ''), NEW."createdAt")
    ON CONFLICT (id) DO UPDATE SET
        email = EXCLUDED.email,
        name = EXCLUDED.name;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS on_auth_user_created ON neon_auth.user;
CREATE TRIGGER on_auth_user_created
    AFTER INSERT ON neon_auth.user
    FOR EACH ROW
EXECUTE FUNCTION public.sync_user_from_auth();

DROP TRIGGER IF EXISTS on_auth_user_updated ON neon_auth.user;
CREATE TRIGGER on_auth_user_updated
    AFTER UPDATE ON neon_auth.user
    FOR EACH ROW
EXECUTE FUNCTION public.sync_user_from_auth();