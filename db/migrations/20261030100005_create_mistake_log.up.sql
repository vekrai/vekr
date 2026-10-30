CREATE TABLE mistake_log (
    id         BIGSERIAL PRIMARY KEY,
    attempt_id BIGINT NOT NULL REFERENCES attempts(id),
    concept_id BIGINT NOT NULL REFERENCES concepts(id),
    error_type TEXT NOT NULL CHECK (error_type IN ('misconception', 'careless_slip', 'knowledge_gap', 'calculation_error')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
