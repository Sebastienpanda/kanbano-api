-- Demande utilisateur : trier automatiquement les tâches d'une colonne par
-- statut (À faire, En cours, Terminé) et renumeroter "position" en
-- conséquence à chaque changement de statut, pour que les tâches terminées
-- passent en dernier sans intervention du front.

CREATE OR REPLACE FUNCTION task_status_rank(p_status task_status)
RETURNS SMALLINT AS $$
BEGIN
    RETURN CASE p_status
        WHEN 'À faire' THEN 0
        WHEN 'En cours' THEN 1
        WHEN 'Terminé' THEN 2
    END;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE OR REPLACE FUNCTION tasks_resequence_column(p_column_id UUID)
RETURNS VOID AS $$
BEGIN
    WITH ordered AS (
        SELECT id,
               ROW_NUMBER() OVER (
                   ORDER BY task_status_rank(status), position, created_at
               ) - 1 AS new_position
        FROM tasks
        WHERE column_id = p_column_id
          AND deleted_at IS NULL
    )
    UPDATE tasks t
    SET position = ordered.new_position
    FROM ordered
    WHERE t.id = ordered.id
      AND t.position IS DISTINCT FROM ordered.new_position;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION tasks_resequence_column_on_status_change()
RETURNS TRIGGER AS $$
BEGIN
    PERFORM tasks_resequence_column(NEW.column_id);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- AFTER UPDATE : le statut a changé, on retrie toute la colonne.
CREATE TRIGGER tasks_resequence_after_status_update
    AFTER UPDATE OF status ON tasks
    FOR EACH ROW
    WHEN (OLD.status IS DISTINCT FROM NEW.status)
    EXECUTE FUNCTION tasks_resequence_column_on_status_change();

-- AFTER INSERT : une tâche est créée (toujours "À faire" via 000041), on
-- s'assure qu'elle reste bien groupée avant les tâches "En cours"/"Terminé".
CREATE TRIGGER tasks_resequence_after_insert
    AFTER INSERT ON tasks
    FOR EACH ROW
    WHEN (NEW.deleted_at IS NULL)
    EXECUTE FUNCTION tasks_resequence_column_on_status_change();
