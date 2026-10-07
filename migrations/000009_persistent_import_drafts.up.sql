ALTER TYPE financial_event_source ADD VALUE 'import';

CREATE TABLE import_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
    source_id TEXT NOT NULL,
    payload_hash CHAR(64) NOT NULL,
    snapshot BYTEA NOT NULL CHECK (octet_length(snapshot) <= 16777216),
    report JSONB NOT NULL,
    normalizer_version TEXT NOT NULL DEFAULT 'grace.import-review.v1',
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
    status TEXT NOT NULL DEFAULT 'reviewing' CHECK (status IN ('reviewing','completed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(id,owner_id), UNIQUE(owner_id,payload_hash)
);
CREATE TABLE import_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
    batch_id UUID NOT NULL,
    snapshot_record_id TEXT NOT NULL,
    domain TEXT NOT NULL,
    source JSONB NOT NULL,
    raw_hash CHAR(64) NOT NULL,
    decision JSONB NOT NULL DEFAULT '{"action":"unresolved"}',
    published BOOLEAN NOT NULL DEFAULT false,
    UNIQUE(id,owner_id), UNIQUE(id,batch_id,owner_id),
    UNIQUE(owner_id,batch_id,snapshot_record_id),
    FOREIGN KEY(batch_id,owner_id) REFERENCES import_batches(id,owner_id)
);
CREATE TABLE import_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
    record_id UUID NOT NULL,
    actor_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
    revision BIGINT NOT NULL,
    decision JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK(actor_id = owner_id),
    FOREIGN KEY(record_id,owner_id) REFERENCES import_records(id,owner_id)
);
CREATE TABLE import_previews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
    batch_id UUID NOT NULL,
    revision BIGINT NOT NULL,
    content JSONB NOT NULL,
    content_hash CHAR(64) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    UNIQUE(id,owner_id), UNIQUE(id,batch_id,owner_id),
    FOREIGN KEY(batch_id,owner_id) REFERENCES import_batches(id,owner_id)
);
CREATE TABLE import_confirmations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
    batch_id UUID NOT NULL,
    preview_id UUID NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash CHAR(64) NOT NULL,
    receipt JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(id,owner_id), UNIQUE(id,batch_id,owner_id), UNIQUE(owner_id,idempotency_key),
    FOREIGN KEY(batch_id,owner_id) REFERENCES import_batches(id,owner_id),
    FOREIGN KEY(preview_id,batch_id,owner_id) REFERENCES import_previews(id,batch_id,owner_id)
);
CREATE TABLE import_provenance (
    record_id UUID NOT NULL,
    batch_id UUID NOT NULL,
    owner_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
    confirmation_id UUID NOT NULL,
    event_id UUID,
    action TEXT NOT NULL CHECK(action IN ('create','link_existing','duplicate_of','exclude')),
    decision JSONB NOT NULL,
    PRIMARY KEY(record_id,owner_id),
    CHECK((action = 'exclude') = (event_id IS NULL)),
    FOREIGN KEY(record_id,batch_id,owner_id) REFERENCES import_records(id,batch_id,owner_id),
    FOREIGN KEY(confirmation_id,batch_id,owner_id) REFERENCES import_confirmations(id,batch_id,owner_id),
    FOREIGN KEY(event_id,owner_id) REFERENCES financial_events(id,owner_id)
);
CREATE TABLE import_requests (
    owner_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
    scope TEXT NOT NULL,
    idempotency_key TEXT NOT NULL CHECK(length(idempotency_key) BETWEEN 1 AND 160),
    request_hash CHAR(64) NOT NULL,
    response JSONB NOT NULL,
    PRIMARY KEY(owner_id,scope,idempotency_key)
);
CREATE INDEX import_records_batch_idx ON import_records(owner_id,batch_id,id);
CREATE INDEX import_records_raw_idx ON import_records(owner_id,raw_hash) WHERE published;
CREATE INDEX import_decisions_record_idx ON import_decisions(owner_id,record_id,revision);
DO $$ DECLARE t TEXT; BEGIN
    FOREACH t IN ARRAY ARRAY['import_batches','import_records','import_decisions',
        'import_previews','import_confirmations','import_provenance','import_requests'] LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY',t);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY',t);
        EXECUTE format('CREATE POLICY owner_access ON %I TO grace_financial USING(owner_id=grace_owner_id()) WITH CHECK(owner_id=grace_owner_id())',t);
        EXECUTE format('GRANT SELECT, INSERT ON %I TO grace_financial',t);
    END LOOP;
    GRANT UPDATE(revision,status) ON import_batches TO grace_financial;
    GRANT UPDATE(decision,published) ON import_records TO grace_financial;
    GRANT UPDATE(receipt) ON import_confirmations TO grace_financial;
END $$;
CREATE FUNCTION grace_import_receipt_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.receipt <> '{}'::jsonb THEN RAISE EXCEPTION 'Confirmation is immutable'; END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER import_receipt_immutable BEFORE UPDATE ON import_confirmations
FOR EACH ROW EXECUTE FUNCTION grace_import_receipt_immutable();
CREATE FUNCTION grace_import_published_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.published THEN RAISE EXCEPTION 'Published source decision is immutable'; END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER import_published_immutable BEFORE UPDATE ON import_records
FOR EACH ROW EXECUTE FUNCTION grace_import_published_immutable();
