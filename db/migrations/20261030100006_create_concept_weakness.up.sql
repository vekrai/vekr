CREATE TABLE concept_weakness (
    concept_id     BIGINT PRIMARY KEY REFERENCES concepts(id),
    miss_count     INT NOT NULL DEFAULT 0,
    attempt_count  INT NOT NULL DEFAULT 0,
    last_missed_at TIMESTAMPTZ
);
