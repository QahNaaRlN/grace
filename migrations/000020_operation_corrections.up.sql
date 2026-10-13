-- SPEC-0041: operations are corrected in place with an append-only history,
-- and voided by moving their postings into an archive. Every balance and
-- analytics query reads `postings`, so a voided operation stops counting
-- everywhere at once while the event and its money stay on record.
ALTER TABLE financial_events
    ADD COLUMN revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    ADD COLUMN edited_by_user BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN voided_at TIMESTAMPTZ,
    ADD COLUMN void_reason TEXT CHECK (void_reason IS NULL OR char_length(void_reason) BETWEEN 1 AND 500),
    ADD CONSTRAINT void_reason_with_void CHECK ((voided_at IS NULL) = (void_reason IS NULL));

CREATE TABLE financial_event_revisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL REFERENCES users (id),
    event_id UUID NOT NULL,
    revision INTEGER NOT NULL CHECK (revision > 1),
    changes JSONB NOT NULL CHECK (jsonb_typeof(changes) = 'array'),
    idempotency_key TEXT NOT NULL CHECK (char_length(idempotency_key) BETWEEN 1 AND 160),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (event_id, revision),
    UNIQUE (owner_id, idempotency_key),
    FOREIGN KEY (event_id, owner_id) REFERENCES financial_events (id, owner_id)
);

CREATE TABLE voided_postings (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL REFERENCES users (id),
    financial_event_id UUID NOT NULL,
    account_id UUID NOT NULL,
    direction posting_direction NOT NULL,
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    currency CHAR(3) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    voided_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (financial_event_id, owner_id) REFERENCES financial_events (id, owner_id),
    FOREIGN KEY (account_id, owner_id) REFERENCES accounts (id, owner_id)
);
CREATE INDEX voided_postings_event_idx ON voided_postings (financial_event_id);

DO $$
DECLARE t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['financial_event_revisions', 'voided_postings'] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('CREATE POLICY owner_access ON %I TO grace_financial USING (owner_id = grace_owner_id()) WITH CHECK (owner_id = grace_owner_id())', t);
        EXECUTE format('GRANT SELECT, INSERT ON %I TO grace_financial', t);
    END LOOP;
END $$;

-- Postings may be deleted only once their own event is voided (they move to
-- voided_postings in the same transaction).
GRANT DELETE ON postings TO grace_financial;
CREATE POLICY delete_voided_only ON postings AS RESTRICTIVE FOR DELETE TO grace_financial
    USING (EXISTS (SELECT 1 FROM financial_events e WHERE e.id = financial_event_id AND e.voided_at IS NOT NULL));

-- A voided operation is final: nothing about it changes afterwards.
CREATE FUNCTION grace_voided_event_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.voided_at IS NOT NULL THEN
        RAISE EXCEPTION 'voided financial event is immutable' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER voided_event_immutable BEFORE UPDATE ON financial_events
    FOR EACH ROW EXECUTE FUNCTION grace_voided_event_immutable();

-- A voided event keeps no live postings and cannot get new ones.
CREATE FUNCTION grace_no_postings_for_voided() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM financial_events WHERE id = NEW.financial_event_id AND voided_at IS NOT NULL) THEN
        RAISE EXCEPTION 'voided financial event cannot have postings' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER no_postings_for_voided BEFORE INSERT OR UPDATE ON postings
    FOR EACH ROW EXECUTE FUNCTION grace_no_postings_for_voided();
