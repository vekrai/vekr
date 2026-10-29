CREATE INDEX nodes_fts_idx ON nodes USING GIN (to_tsvector('english', title || ' ' || content));
