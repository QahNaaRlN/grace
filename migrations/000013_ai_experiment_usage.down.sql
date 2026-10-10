DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM ai_usage) THEN RAISE EXCEPTION 'Cannot discard AI usage evidence'; END IF;
END $$;
DROP FUNCTION grace_ai_reserve(BIGINT,TEXT,TEXT,TEXT),grace_ai_settle(UUID,BIGINT,TEXT),grace_ai_remaining();
DROP TABLE ai_usage,ai_experiment_budget;
