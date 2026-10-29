ALTER TABLE nodes ADD COLUMN type TEXT NOT NULL DEFAULT 'note' CHECK (type IN ('note', 'topic'));
ALTER TABLE nodes ADD COLUMN path_id BIGINT REFERENCES paths(id);
ALTER TABLE nodes ADD COLUMN status TEXT CHECK (status IN ('not_started', 'learning', 'practicing', 'mastered'));
