CREATE TABLE attempts (
    id               BIGSERIAL PRIMARY KEY,
    practice_item_id BIGINT NOT NULL REFERENCES practice_items(id),
    submission       TEXT NOT NULL,
    passed           BOOLEAN,
    agent_feedback   TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
