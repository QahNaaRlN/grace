-- SPEC-0026, stage A. Disabled until an operator records the purchased allowance.
CREATE TABLE ai_experiment_budget (
    singleton BOOLEAN PRIMARY KEY DEFAULT true CHECK(singleton),
    allowance_micros BIGINT NOT NULL DEFAULT 0 CHECK(allowance_micros>=0),
    fees_micros BIGINT NOT NULL DEFAULT 0 CHECK(fees_micros>=0),
    CHECK(allowance_micros+fees_micros<=10000000)
);
INSERT INTO ai_experiment_budget DEFAULT VALUES;
CREATE TABLE ai_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
    reserved_micros BIGINT NOT NULL CHECK(reserved_micros>0 AND reserved_micros<=10000000),
    actual_micros BIGINT CHECK(actual_micros>=0),
    status TEXT NOT NULL DEFAULT 'reserved' CHECK(status IN ('reserved','settled','uncertain')),
    model TEXT NOT NULL,
    provider TEXT NOT NULL,
    purpose TEXT NOT NULL CHECK(purpose IN ('analysis','receipt','jev_category')),
    generation_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    settled_at TIMESTAMPTZ,
    UNIQUE(id,owner_id),
    CHECK((status='settled')=(actual_micros IS NOT NULL))
);
ALTER TABLE ai_usage ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai_usage FORCE ROW LEVEL SECURITY;
CREATE POLICY owner_access ON ai_usage TO grace_financial
USING(owner_id=grace_owner_id()) WITH CHECK(owner_id=grace_owner_id());
GRANT SELECT ON ai_usage TO grace_financial;

-- Only metering, never financial records, crosses tenants here. The global lock
-- covers all owners and prevents concurrent calls spending the same allowance.
CREATE FUNCTION grace_ai_reserve(amount BIGINT, requested_model TEXT, requested_provider TEXT, requested_purpose TEXT)
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
CREATE FUNCTION grace_ai_settle(reservation UUID, actual BIGINT, generation TEXT)
RETURNS VOID LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,public AS $$
DECLARE current_row public.ai_usage; allowance BIGINT;
BEGIN
 SELECT allowance_micros INTO allowance FROM public.ai_experiment_budget WHERE singleton FOR UPDATE;
 SELECT * INTO current_row FROM public.ai_usage WHERE id=reservation AND owner_id=public.grace_owner_id() FOR UPDATE;
 IF NOT FOUND THEN RAISE EXCEPTION 'ai_reservation_not_found'; END IF;
 IF current_row.status='settled' THEN
   IF current_row.actual_micros IS DISTINCT FROM actual OR current_row.generation_id IS DISTINCT FROM generation THEN RAISE EXCEPTION 'ai_settlement_conflict'; END IF;
   RETURN;
 END IF;
 IF actual IS NOT NULL AND (actual<0 OR generation IS NULL OR length(generation)>200) THEN RAISE EXCEPTION 'invalid_ai_settlement'; END IF;
 -- An upstream overrun is recorded honestly and freezes later calls via the
 -- negative remaining allowance. Missing cost stays reserved and freezes all.
 UPDATE public.ai_usage SET actual_micros=actual,status=CASE WHEN actual IS NULL THEN 'uncertain' ELSE 'settled' END,
 generation_id=generation,settled_at=now() WHERE id=current_row.id;
END $$;
CREATE FUNCTION grace_ai_remaining() RETURNS BIGINT LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog,public AS $$
 SELECT CASE WHEN EXISTS(SELECT 1 FROM public.ai_usage WHERE status='uncertain') THEN 0 ELSE
 greatest(0,b.allowance_micros-COALESCE((SELECT sum(CASE WHEN status='settled' THEN actual_micros ELSE reserved_micros END) FROM public.ai_usage),0)) END
 FROM public.ai_experiment_budget b WHERE singleton;
$$;
REVOKE ALL ON FUNCTION grace_ai_reserve(BIGINT,TEXT,TEXT,TEXT),grace_ai_settle(UUID,BIGINT,TEXT),grace_ai_remaining() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION grace_ai_reserve(BIGINT,TEXT,TEXT,TEXT),grace_ai_settle(UUID,BIGINT,TEXT),grace_ai_remaining() TO grace_financial;
