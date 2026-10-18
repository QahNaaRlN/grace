-- SPEC-0055: separate document-specific consent and aggregate AI reservation.
ALTER TABLE ai_usage DROP CONSTRAINT ai_usage_purpose_check;
ALTER TABLE ai_usage ADD CONSTRAINT ai_usage_purpose_check CHECK(purpose IN ('analysis','receipt','jev_category','statement'));
ALTER TABLE statement_sources ADD COLUMN ai_calls integer NOT NULL DEFAULT 0 CHECK(ai_calls BETWEEN 0 AND 32);
CREATE TABLE statement_extraction_previews (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 statement_id uuid NOT NULL,
 source_revision bigint NOT NULL,
 state_hash char(64) NOT NULL,
 plan_hash char(64) NOT NULL,
 plan jsonb,
 policy jsonb NOT NULL,
 prompt_version text NOT NULL,
 schema_hash char(64) NOT NULL,
 redaction_version text NOT NULL,
 maximum_micros bigint NOT NULL CHECK(maximum_micros BETWEEN 1 AND 256000),
 call_count integer NOT NULL CHECK(call_count BETWEEN 1 AND 32),
 created_at timestamptz NOT NULL DEFAULT now(),
 expires_at timestamptz NOT NULL DEFAULT now()+interval '15 minutes',
 UNIQUE(id,owner_id),UNIQUE(id,statement_id,owner_id),
 FOREIGN KEY(statement_id,owner_id) REFERENCES statement_sources(id,owner_id)
);
CREATE TABLE statement_extraction_consents (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 statement_id uuid NOT NULL,
 preview_id uuid NOT NULL,
 preview_hash char(64) NOT NULL,
 creation_key text NOT NULL CHECK(length(creation_key) BETWEEN 1 AND 160),
 created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(id,owner_id),UNIQUE(id,preview_id,statement_id,owner_id),UNIQUE(owner_id,creation_key),
 FOREIGN KEY(preview_id,statement_id,owner_id) REFERENCES statement_extraction_previews(id,statement_id,owner_id)
);
CREATE TABLE statement_extraction_attempts (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 statement_id uuid NOT NULL,
 preview_id uuid NOT NULL,
 consent_id uuid NOT NULL,
 source_revision bigint NOT NULL,
 reservation_id uuid NOT NULL,
 idempotency_key text NOT NULL CHECK(length(idempotency_key) BETWEEN 1 AND 160),
 request_hash char(64) NOT NULL,
 status text NOT NULL DEFAULT 'running' CHECK(status IN ('running','completed','failed','uncertain','cancelled')),
 error_code text,
 created_at timestamptz NOT NULL DEFAULT now(),
 finished_at timestamptz,
 UNIQUE(id,owner_id),UNIQUE(id,statement_id,owner_id),UNIQUE(owner_id,statement_id,idempotency_key),
 FOREIGN KEY(statement_id,owner_id) REFERENCES statement_sources(id,owner_id),
 FOREIGN KEY(preview_id,statement_id,owner_id) REFERENCES statement_extraction_previews(id,statement_id,owner_id),
 FOREIGN KEY(consent_id,preview_id,statement_id,owner_id) REFERENCES statement_extraction_consents(id,preview_id,statement_id,owner_id),
 FOREIGN KEY(reservation_id,owner_id) REFERENCES ai_usage(id,owner_id)
);
CREATE UNIQUE INDEX statement_one_running_attempt_idx ON statement_extraction_attempts(statement_id,owner_id) WHERE status='running';
CREATE TABLE statement_extraction_calls (
 attempt_id uuid NOT NULL,
 statement_id uuid NOT NULL,
 owner_id uuid NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
 call_index integer NOT NULL CHECK(call_index BETWEEN 1 AND 32),
 status text NOT NULL DEFAULT 'started' CHECK(status IN ('started','completed','uncertain')),
 generation_id text CHECK(length(generation_id)<=200),
 cost_micros bigint CHECK(cost_micros>=0),
 input_tokens integer,
 output_tokens integer,
 started_at timestamptz NOT NULL DEFAULT now(),
 finished_at timestamptz,
 PRIMARY KEY(attempt_id,call_index,owner_id),
 FOREIGN KEY(attempt_id,statement_id,owner_id) REFERENCES statement_extraction_attempts(id,statement_id,owner_id),
 CHECK((status='completed')=(cost_micros IS NOT NULL))
);
DO $$ DECLARE t text; BEGIN
 FOREACH t IN ARRAY ARRAY['statement_extraction_previews','statement_extraction_consents','statement_extraction_attempts','statement_extraction_calls'] LOOP
  EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY',t);
  EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY',t);
  EXECUTE format('CREATE POLICY owner_access ON %I TO grace_financial USING(owner_id=grace_owner_id()) WITH CHECK(owner_id=grace_owner_id())',t);
  EXECUTE format('GRANT SELECT,INSERT ON %I TO grace_financial',t);
 END LOOP;
END $$;
GRANT UPDATE(plan) ON statement_extraction_previews TO grace_financial;
GRANT UPDATE(status,error_code,finished_at) ON statement_extraction_attempts TO grace_financial;
GRANT UPDATE(status,generation_id,cost_micros,input_tokens,output_tokens,finished_at) ON statement_extraction_calls TO grace_financial;
GRANT UPDATE(ai_calls) ON statement_sources TO grace_financial;
CREATE FUNCTION grace_statement_call_counter_guard() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NEW.ai_calls<OLD.ai_calls OR NEW.ai_calls>OLD.ai_calls+1 THEN RAISE EXCEPTION 'Statement calls are monotonic'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER statement_call_counter_guard BEFORE UPDATE ON statement_sources FOR EACH ROW EXECUTE FUNCTION grace_statement_call_counter_guard();
CREATE FUNCTION grace_statement_clear_extraction() RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public AS $$ BEGIN
 IF NEW.kind='bank_statement' AND NEW.content IS NULL AND OLD.content IS NOT NULL THEN
  UPDATE public.statement_extraction_previews SET plan=NULL WHERE statement_id IN (SELECT id FROM public.statement_sources WHERE document_id=NEW.id AND owner_id=NEW.owner_id) AND owner_id=NEW.owner_id;
  UPDATE public.statement_extraction_attempts SET status='cancelled',error_code='statement_original_gone',finished_at=now() WHERE statement_id IN (SELECT id FROM public.statement_sources WHERE document_id=NEW.id AND owner_id=NEW.owner_id) AND owner_id=NEW.owner_id AND status='running';
 END IF;
 RETURN NEW;
END $$;
REVOKE ALL ON FUNCTION grace_statement_clear_extraction() FROM PUBLIC;
CREATE TRIGGER statement_clear_extraction AFTER UPDATE ON documents FOR EACH ROW EXECUTE FUNCTION grace_statement_clear_extraction();

CREATE OR REPLACE FUNCTION grace_ai_reserve(amount BIGINT, requested_model TEXT, requested_provider TEXT, requested_purpose TEXT)
RETURNS UUID LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public AS $$
DECLARE allowance BIGINT; committed BIGINT; result UUID; owner UUID;
BEGIN
 owner:=public.grace_owner_id();
 IF owner IS NULL OR amount<=0 OR amount>10000000 THEN RAISE EXCEPTION 'invalid_ai_reservation'; END IF;
 SELECT allowance_micros INTO allowance FROM public.ai_experiment_budget WHERE singleton FOR UPDATE;
 UPDATE public.ai_usage SET status='uncertain' WHERE status='reserved' AND created_at<now()-interval '3 minutes';
 UPDATE public.ai_usage u SET status='uncertain' WHERE u.status='reserved' AND u.purpose='statement' AND EXISTS(SELECT 1 FROM public.statement_extraction_calls c JOIN public.statement_extraction_attempts a ON a.id=c.attempt_id AND a.owner_id=c.owner_id WHERE a.reservation_id=u.id AND c.status='started' AND c.started_at<now()-interval '35 seconds');
 IF EXISTS(SELECT 1 FROM public.ai_usage WHERE status='uncertain') THEN RAISE EXCEPTION 'ai_usage_requires_review'; END IF;
 IF (SELECT count(*) FROM public.ai_usage WHERE owner_id=owner AND status='reserved')>=2 THEN RAISE EXCEPTION 'ai_concurrency_limit'; END IF;
 SELECT COALESCE(sum(CASE WHEN status='settled' THEN actual_micros ELSE reserved_micros END),0) INTO committed FROM public.ai_usage;
 IF amount>allowance-committed THEN RAISE EXCEPTION 'budget_exhausted'; END IF;
 INSERT INTO public.ai_usage(owner_id,reserved_micros,model,provider,purpose)
 VALUES(owner,amount,requested_model,requested_provider,requested_purpose) RETURNING id INTO result;
 RETURN result;
END $$;

CREATE FUNCTION grace_statement_call_allowed(reservation uuid) RETURNS boolean
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public AS $$
DECLARE allowance bigint;committed bigint;
BEGIN
 SELECT allowance_micros INTO allowance FROM public.ai_experiment_budget WHERE singleton FOR UPDATE;
 IF NOT EXISTS(SELECT 1 FROM public.ai_usage WHERE id=reservation AND owner_id=public.grace_owner_id() AND status='reserved') OR EXISTS(SELECT 1 FROM public.ai_usage WHERE status='uncertain') THEN RETURN false; END IF;
 SELECT COALESCE(sum(CASE WHEN status='settled' THEN actual_micros ELSE reserved_micros END),0) INTO committed FROM public.ai_usage;
 RETURN committed<=allowance;
END $$;
REVOKE ALL ON FUNCTION grace_statement_call_allowed(uuid) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION grace_statement_call_allowed(uuid) TO grace_financial;
