-- À la création d'un utilisateur (sync depuis Neon Auth), on initialise son
-- espace : organisation + adhésion + workspace de démarrage (3 colonnes, et
-- 3 tâches dans la première colonne).
--
-- Le bloc d'init ne s'exécute QUE lorsqu'une nouvelle organisation est créée
-- (RETURNING renvoie NULL sur conflit), donc jamais lors des re-syncs d'un
-- utilisateur déjà connu.

CREATE OR REPLACE FUNCTION public.sync_user_from_auth()
    RETURNS TRIGGER
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = public
AS $$
DECLARE
    v_org_id       UUID;
    v_workspace_id UUID;
    v_column_id    UUID;
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
    ON CONFLICT (user_id) DO NOTHING
    RETURNING id INTO v_org_id;

    -- Organisation déjà existante : rien à initialiser.
    IF v_org_id IS NULL THEN
        RETURN NEW;
    END IF;

    INSERT INTO public.organisation_members (member_id, organisation_id)
    VALUES (NEW.id, v_org_id);

    INSERT INTO public.workspaces (name, created_by, organisation_id)
    VALUES ('Mon premier workspace', NEW.id, v_org_id)
    RETURNING id INTO v_workspace_id;

    INSERT INTO public.columns (name, position, workspace_id, created_by)
    VALUES ('À faire',  0, v_workspace_id, NEW.id),
           ('En cours', 1, v_workspace_id, NEW.id),
           ('Terminé',  2, v_workspace_id, NEW.id);

    SELECT id INTO v_column_id
    FROM public.columns
    WHERE workspace_id = v_workspace_id
      AND position = 0;

    INSERT INTO public.tasks (name, position, column_id, created_by)
    VALUES ('Première tâche',  0, v_column_id, NEW.id),
           ('Deuxième tâche',  1, v_column_id, NEW.id),
           ('Troisième tâche', 2, v_column_id, NEW.id);

    RETURN NEW;
END;
$$;
