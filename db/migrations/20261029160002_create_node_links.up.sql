CREATE TABLE node_links (
    id           BIGSERIAL PRIMARY KEY,
    from_node_id BIGINT NOT NULL REFERENCES nodes(id),
    to_node_id   BIGINT NOT NULL REFERENCES nodes(id),
    relation     TEXT NOT NULL CHECK (relation IN ('related', 'prerequisite_of', 'source_of'))
);

CREATE INDEX node_links_from_node_id_idx ON node_links (from_node_id);
CREATE INDEX node_links_to_node_id_idx ON node_links (to_node_id);
