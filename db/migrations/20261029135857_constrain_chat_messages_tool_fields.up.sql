ALTER TABLE chat_messages
    ADD CONSTRAINT chat_messages_tool_call_id_only_on_tool_role
        CHECK (
            (role = 'tool' AND tool_call_id <> '')
            OR (role <> 'tool' AND tool_call_id = '')
        ),
    ADD CONSTRAINT chat_messages_tool_calls_only_on_assistant_role
        CHECK (
            role = 'assistant'
            OR tool_calls = ''
        );

CREATE UNIQUE INDEX chat_messages_chat_id_tool_call_id_idx
    ON chat_messages (chat_id, tool_call_id)
    WHERE tool_call_id <> '';
