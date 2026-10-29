ALTER TABLE nodes ADD COLUMN source_chat_id BIGINT REFERENCES chats(id);
