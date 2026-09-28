-- Une assignation indique seulement qui travaille sur la tâche : elle ne
-- donne ni ne change aucun droit. Le rôle sur une tâche est celui du
-- workspace (ou de l'invitation guest).
ALTER TABLE task_assignees
    DROP COLUMN role;
