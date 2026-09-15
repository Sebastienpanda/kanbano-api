-- Demande utilisateur : le statut "À faire" doit être appliqué par défaut à la
-- création d'une tâche, pour éviter d'oublier de le choisir manuellement.

UPDATE tasks SET status = 'À faire' WHERE status IS NULL;

ALTER TABLE tasks ALTER COLUMN status SET DEFAULT 'À faire';
ALTER TABLE tasks ALTER COLUMN status SET NOT NULL;
