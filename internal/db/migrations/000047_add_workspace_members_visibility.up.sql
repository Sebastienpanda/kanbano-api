-- Un workspace est privé par défaut pour les membres de l'organisation :
-- l'appartenance à l'organisation ne donne plus, à elle seule, accès à ses
-- workspaces. Seuls le créateur du workspace (= propriétaire de
-- l'organisation) et les membres explicitement autorisés via
-- workspace_members (visibility = 'public') y ont accès.
ALTER TABLE workspace_members
    ADD COLUMN visibility TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public'));

ALTER TABLE workspace_members
    ALTER COLUMN role SET DEFAULT 'view';
