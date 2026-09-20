-- Simplification du modèle de droits : la résolution de rôle ne repose plus
-- que sur deux niveaux — organisation_members (rôle de base) et
-- task_assignees (rôle précis sur une tâche, qui l'emporte toujours quand
-- il existe). Les grants à portée workspace/colonne n'ont jamais été
-- exposés par une fonctionnalité produit ("inviter sur une colonne"
-- n'existe pas) : access_grants est donc supprimée.
DROP TABLE IF EXISTS access_grants;
DROP FUNCTION IF EXISTS access_grants_check_column_workspace();
