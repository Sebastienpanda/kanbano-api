-- Avant d'exécuter cette migration en production, vérifier qu'aucune ligne
-- existante ne dépasse 255 caractères sur name (sinon l'ALTER échoue) :
--   SELECT 'workspaces', count(*) FROM workspaces WHERE length(name) > 255
--   UNION ALL SELECT 'columns', count(*) FROM columns WHERE length(name) > 255
--   UNION ALL SELECT 'tasks', count(*) FROM tasks WHERE length(name) > 255;
-- Cette réécriture de colonne prend un ACCESS EXCLUSIVE LOCK sur toute la
-- durée du rewrite ; à exécuter hors des heures de forte charge si les
-- tables sont volumineuses.

ALTER TABLE workspaces ALTER COLUMN name TYPE VARCHAR(255);
ALTER TABLE columns ALTER COLUMN name TYPE VARCHAR(255);
ALTER TABLE tasks ALTER COLUMN name TYPE VARCHAR(255);
