CREATE TABLE practice_item_concepts (
    practice_item_id BIGINT NOT NULL REFERENCES practice_items(id),
    concept_id        BIGINT NOT NULL REFERENCES concepts(id),
    PRIMARY KEY (practice_item_id, concept_id)
);
