DROP INDEX chat_messages_chat_id_tool_call_id_idx;

ALTER TABLE chat_messages
    DROP CONSTRAINT chat_messages_tool_call_id_only_on_tool_role,
    DROP CONSTRAINT chat_messages_tool_calls_only_on_assistant_role;
