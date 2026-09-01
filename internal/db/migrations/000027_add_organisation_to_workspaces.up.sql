-- Rattachement des workspaces à une organisation + table des membres
-- d'organisation.

-- ── workspaces.organisation_id ───────────────────────────────
ALTER TABLE workspaces
    ADD COLUMN organisation_id UUID REFERENCES organisations(id) ON DELETE CASCADE;

-- Backfill : chaque workspace rejoint l'organisation de son créateur.
UPDATE workspaces w
SET organisation_id = o.id
FROM organisations o
WHERE o.user_id = w.created_by;

ALTER TABLE workspaces ALTER COLUMN organisation_id SET NOT NULL;

-- ── organisation_members ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS organisation_members (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organisation_id UUID NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,
    UNIQUE (member_id, organisation_id)
);

-- Backfill : le propriétaire de chaque organisation en est membre.
INSERT INTO organisation_members (member_id, organisation_id)
SELECT user_id, id FROM organisations
ON CONFLICT (member_id, organisation_id) DO NOTHING;
