-- Invitations d'invités externes, scopées à une seule tâche. Contrairement à
-- task_assignees (réservée aux membres d'organisation, cf. organisation_members),
-- un invité externe n'appartient à aucune organisation : il ne voit que le
-- workspace en lecture seule et sa tâche selon le rôle qui lui a été accordé.
-- Comme organisation_invitations, l'invité peut ne pas encore avoir de compte
-- Kanbano : la ligne reste identifiée par email jusqu'à acceptation, où
-- user_id est renseigné.

CREATE TABLE IF NOT EXISTS task_guests (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id      UUID NOT NULL REFERENCES tasks(id),
    email        TEXT NOT NULL
                     CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    user_id      UUID REFERENCES users(id),
    role         TEXT NOT NULL DEFAULT 'view' CHECK (role IN ('view', 'edit')),
    status       TEXT NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'accepted', 'declined')),
    invited_by   UUID NOT NULL REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    responded_at TIMESTAMPTZ
);

-- Une seule invitation pending à la fois pour un couple (task, email).
CREATE UNIQUE INDEX task_guests_pending_unique
    ON task_guests (task_id, LOWER(email))
    WHERE status = 'pending';

CREATE INDEX task_guests_email_idx ON task_guests (LOWER(email));
CREATE INDEX task_guests_user_id_idx ON task_guests (user_id) WHERE user_id IS NOT NULL;
CREATE INDEX task_guests_task_id_idx ON task_guests (task_id);
