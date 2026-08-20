ALTER TABLE tasks ADD COLUMN state_id UUID REFERENCES states(id);
