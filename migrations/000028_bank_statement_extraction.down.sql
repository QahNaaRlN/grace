DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM statement_extraction_previews) OR EXISTS(SELECT 1 FROM statement_extraction_consents) OR EXISTS(SELECT 1 FROM statement_extraction_attempts) OR EXISTS(SELECT 1 FROM statement_extraction_calls) OR EXISTS(SELECT 1 FROM ai_usage WHERE purpose='statement')
 THEN RAISE EXCEPTION 'Cannot downgrade statement consent or billing evidence'; END IF;
END $$;
DROP FUNCTION grace_statement_call_allowed(uuid);
CREATE OR REPLACE FUNCTION grace_ai_reserve(amount BIGINT, requested_model TEXT, requested_provider TEXT, requested_purpose TEXT)
RETURNS UUID LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public AS $$
DECLARE allowance BIGINT; committed BIGINT; result UUID; owner UUID;
BEGIN
 owner:=public.grace_owner_id();
 IF owner IS NULL OR amount<=0 OR amount>10000000 THEN RAISE EXCEPTION 'invalid_ai_reservation'; END IF;
 SELECT allowance_micros INTO allowance FROM public.ai_experiment_budget WHERE singleton FOR UPDATE;
 UPDATE public.ai_usage SET status='uncertain' WHERE status='reserved' AND created_at<now()-interval '3 minutes';
 IF EXISTS(SELECT 1 FROM public.ai_usage WHERE status='uncertain') THEN RAISE EXCEPTION 'ai_usage_requires_review'; END IF;
 IF (SELECT count(*) FROM public.ai_usage WHERE owner_id=owner AND status='reserved')>=2 THEN RAISE EXCEPTION 'ai_concurrency_limit'; END IF;
 SELECT COALESCE(sum(CASE WHEN status='settled' THEN actual_micros ELSE reserved_micros END),0) INTO committed FROM public.ai_usage;
 IF amount>allowance-committed THEN RAISE EXCEPTION 'budget_exhausted'; END IF;
 INSERT INTO public.ai_usage(owner_id,reserved_micros,model,provider,purpose)
 VALUES(owner,amount,requested_model,requested_provider,requested_purpose) RETURNING id INTO result;
 RETURN result;
END $$;
DROP TRIGGER statement_clear_extraction ON documents;
DROP FUNCTION grace_statement_clear_extraction();
DROP TRIGGER statement_call_counter_guard ON statement_sources;
DROP FUNCTION grace_statement_call_counter_guard();
DROP TABLE statement_extraction_calls;
DROP TABLE statement_extraction_attempts;
DROP TABLE statement_extraction_consents;
DROP TABLE statement_extraction_previews;
REVOKE UPDATE(ai_calls) ON statement_sources FROM grace_financial;
ALTER TABLE statement_sources DROP COLUMN ai_calls;
ALTER TABLE ai_usage DROP CONSTRAINT ai_usage_purpose_check;
ALTER TABLE ai_usage ADD CONSTRAINT ai_usage_purpose_check CHECK(purpose IN ('analysis','receipt','jev_category'));
