-- Invitations guest : expiration au bout de 7 jours (repoussée quand
-- l'inviteur la renvoie), et un seul accès accepté par (tâche, utilisateur).
--
-- Avant d'exécuter en production, vérifier qu'aucun doublon accepté n'existe
-- (sinon la création de l'index échoue) :
--   SELECT task_id, user_id, count(*) FROM task_guests
--   WHERE status = 'accepted' GROUP BY task_id, user_id HAVING count(*) > 1;

ALTER TABLE task_guests
    ADD COLUMN expires_at TIMESTAMPTZ;

UPDATE task_guests
SET expires_at = created_at + INTERVAL '7 days';

ALTER TABLE task_guests
    ALTER COLUMN expires_at SET DEFAULT NOW() + INTERVAL '7 days',
    ALTER COLUMN expires_at SET NOT NULL;

CREATE UNIQUE INDEX task_guests_accepted_unique
    ON task_guests (task_id, user_id)
    WHERE status = 'accepted';
