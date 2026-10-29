ALTER TABLE chat_messages
    ADD COLUMN tool_call_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN tool_calls   TEXT NOT NULL DEFAULT '',
    ADD COLUMN name         TEXT NOT NULL DEFAULT '';
