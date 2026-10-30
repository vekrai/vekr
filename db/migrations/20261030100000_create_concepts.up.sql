CREATE TABLE concepts (
    id      BIGSERIAL PRIMARY KEY,
    path_id BIGINT REFERENCES paths(id),
    name    TEXT NOT NULL
);
