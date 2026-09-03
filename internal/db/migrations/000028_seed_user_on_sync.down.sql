-- Retour à la version de la fonction issue de 000024 (users + organisations,
-- sans initialisation du workspace de démarrage).

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

    INSERT INTO public.organisations (user_id)
    VALUES (NEW.id)
    ON CONFLICT (user_id) DO NOTHING;

    RETURN NEW;
END;
$$;
