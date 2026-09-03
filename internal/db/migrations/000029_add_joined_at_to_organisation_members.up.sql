-- Date d'ajout d'un membre à une organisation (NULL pour les membres seedés).
ALTER TABLE organisation_members
    ADD COLUMN joined_at TIMESTAMPTZ;
