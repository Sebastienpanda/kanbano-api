-- WorkspaceRepository.Search fait un ILIKE '%...%' sur name/description sans
-- index exploitable, donc un scan séquentiel à chaque recherche. pg_trgm
-- permet un index GIN utilisable par ILIKE avec motif libre.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX workspaces_name_trgm_idx ON workspaces USING gin (name gin_trgm_ops);
CREATE INDEX workspaces_description_trgm_idx ON workspaces USING gin (description gin_trgm_ops);
