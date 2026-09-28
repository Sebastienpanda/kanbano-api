-- Organisations : un nom modifiable, et l'owner n'est plus stocké dans
-- organisation_members. L'owner reste déterminé par organisations.user_id ;
-- sa ligne organisation_members (insérée par 000028, passée à 'member' par
-- 000048) le faisait apparaître comme simple membre.

-- ── organisations.name ───────────────────────────────────────
ALTER TABLE organisations
    ADD COLUMN name VARCHAR(255);

UPDATE organisations o
SET name = LEFT('Organisation de ' || COALESCE(NULLIF(u.name, ''), SPLIT_PART(u.email, '@', 1)), 255)
FROM users u
WHERE u.id = o.user_id;

ALTER TABLE organisations
    ALTER COLUMN name SET NOT NULL;

-- ── Suppression douce ────────────────────────────────────────
-- Supprimer une organisation la masque avec ses workspaces, colonnes et
-- tâches (même mécanisme que les workspaces) : rien n'est effacé en base.
-- La ligne garde organisations.user_id : l'utilisateur ne redevient pas
-- owner d'une autre organisation.
ALTER TABLE organisations
    ADD COLUMN deleted_at TIMESTAMPTZ,
    ADD COLUMN deleted_by UUID REFERENCES users(id);

-- ── organisation_members : retrait des owners ────────────────
DELETE FROM organisation_members om
USING organisations o
WHERE o.id = om.organisation_id
  AND o.user_id = om.member_id;

-- ── Trigger d'inscription ────────────────────────────────────
-- Même initialisation que 000028, avec le nom de l'organisation et sans
-- ligne organisation_members pour l'owner.
CREATE OR REPLACE FUNCTION public.sync_user_from_auth()
    RETURNS TRIGGER
    LANGUAGE plpgsql
    SECURITY DEFINER
    SET search_path = public
AS $$
DECLARE
    v_name         TEXT;
    v_org_id       UUID;
    v_workspace_id UUID;
    v_column_id    UUID;
BEGIN
    v_name := COALESCE(NULLIF(NEW.name, ''), SPLIT_PART(NEW.email, '@', 1));

    INSERT INTO public.users (id, email, name, created_at)
    VALUES (NEW.id, NEW.email, v_name, NEW."createdAt")
    ON CONFLICT (id) DO UPDATE SET
        email = EXCLUDED.email,
        name = EXCLUDED.name;

    INSERT INTO public.organisations (user_id, name)
    VALUES (NEW.id, LEFT('Organisation de ' || v_name, 255))
    ON CONFLICT (user_id) DO NOTHING
    RETURNING id INTO v_org_id;

    -- Organisation déjà existante : rien à initialiser.
    IF v_org_id IS NULL THEN
        RETURN NEW;
    END IF;

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
