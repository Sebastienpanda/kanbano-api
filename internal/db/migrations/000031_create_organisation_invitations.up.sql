-- Invitations à rejoindre une organisation, envoyées par email. L'invité
-- peut ne pas encore avoir de compte Kanbano : l'invitation reste "pending"
-- jusqu'à ce qu'il l'accepte (après connexion, quel que soit l'ordre).

CREATE TABLE IF NOT EXISTS organisation_invitations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id UUID NOT NULL REFERENCES organisations(id) ON DELETE CASCADE,
    email           TEXT NOT NULL,
    invited_by      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status          TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'accepted', 'declined')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    responded_at    TIMESTAMPTZ
);

-- Une seule invitation pending à la fois pour un couple (organisation, email).
CREATE UNIQUE INDEX organisation_invitations_pending_unique
    ON organisation_invitations (organisation_id, LOWER(email))
    WHERE status = 'pending';

CREATE INDEX organisation_invitations_email_idx
    ON organisation_invitations (LOWER(email));
