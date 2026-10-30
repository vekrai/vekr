CREATE TABLE practice_items (
    id         BIGSERIAL PRIMARY KEY,
    node_id    BIGINT NOT NULL REFERENCES nodes(id),
    prompt     TEXT NOT NULL,
    answer_key TEXT,
    type       TEXT NOT NULL CHECK (type IN ('mcq', 'flashcard', 'open', 'code')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
