CREATE TABLE chats (
    id         BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE chat_messages (
    id         BIGSERIAL PRIMARY KEY,
    chat_id    BIGINT NOT NULL REFERENCES chats(id),
    role       TEXT NOT NULL,
    content    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX chat_messages_chat_id_created_at_idx ON chat_messages (chat_id, created_at);
