CREATE TABLE node_concepts (
    node_id    BIGINT NOT NULL REFERENCES nodes(id),
    concept_id BIGINT NOT NULL REFERENCES concepts(id),
    PRIMARY KEY (node_id, concept_id)
);
