CREATE TYPE task_tag AS ENUM ('A faire', 'En cours', 'Terminé');

ALTER TABLE tasks ADD COLUMN tag task_tag;
