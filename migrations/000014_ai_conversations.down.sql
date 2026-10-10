DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM ai_messages) THEN RAISE EXCEPTION 'Cannot discard AI conversation evidence'; END IF;
END $$;
DROP TABLE ai_messages,ai_conversations,ai_consents;
DROP FUNCTION grace_ai_message_complete_immutable();
