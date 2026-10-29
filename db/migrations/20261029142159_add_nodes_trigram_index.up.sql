CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX nodes_trgm_idx ON nodes USING GIN ((title || ' ' || content) gin_trgm_ops);
