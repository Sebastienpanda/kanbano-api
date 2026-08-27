CREATE OR REPLACE FUNCTION public.sync_user_from_auth()
    RETURNS TRIGGER
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = public
AS $$
BEGIN
    INSERT INTO public.users (id, email, name, created_at)
    VALUES (
        NEW.id,
        NEW.email,
        COALESCE(NULLIF(NEW.name, ''), SPLIT_PART(NEW.email, '@', 1)),
        NEW."createdAt"
    )
    ON CONFLICT (id) DO UPDATE SET
        email = EXCLUDED.email,
        name = EXCLUDED.name;
    RETURN NEW;
END;
$$;
