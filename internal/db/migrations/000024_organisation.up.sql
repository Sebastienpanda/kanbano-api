CREATE TABLE IF NOT EXISTS organisations (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES public.users(id) ON DELETE CASCADE
);

-- L'organisation du user est créée en même temps que son compte : on étend le
-- trigger de sync Neon Auth pour insérer aussi la ligne organisations.
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

-- Backfill : une organisation pour chaque utilisateur déjà présent.
INSERT INTO public.organisations (user_id)
SELECT id FROM public.users
ON CONFLICT (user_id) DO NOTHING;
