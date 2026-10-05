CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name TEXT NOT NULL CHECK (length(trim(display_name)) BETWEEN 1 AND 160),
    role TEXT NOT NULL CHECK (role IN ('admin', 'member')),
    disabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO users (id, display_name, role)
VALUES ('00000000-0000-4000-8000-000000000001', 'Владелец', 'admin');

CREATE TABLE auth_identities (
    issuer TEXT NOT NULL,
    subject TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    PRIMARY KEY (issuer, subject),
    UNIQUE (user_id, issuer)
);
CREATE TABLE auth_invitations (
    token_hash CHAR(64) PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ
);
CREATE TABLE auth_login_flows (
    state_hash CHAR(64) PRIMARY KEY,
    browser_hash CHAR(64) NOT NULL,
    verifier TEXT NOT NULL,
    nonce TEXT NOT NULL,
    invitation_hash TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE auth_sessions (
    token_hash CHAR(64) PRIMARY KEY,
    csrf_hash CHAR(64) NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);
CREATE INDEX auth_sessions_user_idx ON auth_sessions(user_id);
CREATE INDEX auth_sessions_expiry_idx ON auth_sessions(expires_at);

CREATE FUNCTION grace_owner_id() RETURNS UUID LANGUAGE sql STABLE AS $$
    SELECT nullif(current_setting('grace.owner_id', true), '')::uuid
$$;

DO $$
DECLARE t TEXT;
BEGIN
    FOREACH t IN ARRAY ARRAY['accounts','financial_events','postings','receipts',
        'receipt_items','obligations','scheduled_payments','scheduled_payment_settlements']
    LOOP
        EXECUTE format('ALTER TABLE %I ADD COLUMN owner_id UUID NOT NULL DEFAULT %L REFERENCES users(id)',
            t, '00000000-0000-4000-8000-000000000001');
        EXECUTE format('ALTER TABLE %I ALTER COLUMN owner_id SET DEFAULT grace_owner_id()', t);
        EXECUTE format('ALTER TABLE %I ADD UNIQUE (id, owner_id)', t);
        EXECUTE format('CREATE INDEX %I ON %I (owner_id)', t || '_owner_idx', t);
    END LOOP;
END $$;

ALTER TABLE financial_events DROP CONSTRAINT financial_events_kind_idempotency_key_key;
ALTER TABLE financial_events ADD UNIQUE(owner_id, kind, idempotency_key);
ALTER TABLE receipts DROP CONSTRAINT receipts_idempotency_key_key;
ALTER TABLE receipts ADD UNIQUE(owner_id, idempotency_key);
ALTER TABLE obligations DROP CONSTRAINT obligations_idempotency_key_key;
ALTER TABLE obligations ADD UNIQUE(owner_id, idempotency_key);
ALTER TABLE scheduled_payment_settlements DROP CONSTRAINT scheduled_payment_settlements_idempotency_key_key;
ALTER TABLE scheduled_payment_settlements ADD UNIQUE(owner_id, idempotency_key);

ALTER TABLE postings ADD CONSTRAINT postings_event_owner_fk
    FOREIGN KEY(financial_event_id, owner_id) REFERENCES financial_events(id, owner_id);
ALTER TABLE postings ADD CONSTRAINT postings_account_owner_fk
    FOREIGN KEY(account_id, owner_id) REFERENCES accounts(id, owner_id);
ALTER TABLE financial_events ADD CONSTRAINT events_original_owner_fk
    FOREIGN KEY(original_event_id, owner_id) REFERENCES financial_events(id, owner_id);
ALTER TABLE receipts ADD CONSTRAINT receipts_purchase_owner_fk
    FOREIGN KEY(purchase_event_id, owner_id) REFERENCES financial_events(id, owner_id);
ALTER TABLE receipt_items ADD CONSTRAINT receipt_items_owner_fk
    FOREIGN KEY(receipt_id, owner_id) REFERENCES receipts(id, owner_id);
ALTER TABLE scheduled_payments ADD CONSTRAINT scheduled_payments_owner_fk
    FOREIGN KEY(obligation_id, owner_id) REFERENCES obligations(id, owner_id);
ALTER TABLE scheduled_payment_settlements ADD CONSTRAINT settlements_payment_owner_fk
    FOREIGN KEY(scheduled_payment_id, owner_id) REFERENCES scheduled_payments(id, owner_id);
ALTER TABLE scheduled_payment_settlements ADD CONSTRAINT settlements_event_owner_fk
    FOREIGN KEY(financial_event_id, owner_id) REFERENCES financial_events(id, owner_id);

DO $$
DECLARE t TEXT;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'grace_financial') THEN
        CREATE ROLE grace_financial NOLOGIN NOSUPERUSER NOBYPASSRLS;
    END IF;
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'grace_financial' AND (rolsuper OR rolbypassrls OR rolcanlogin)) THEN
        RAISE EXCEPTION 'grace_financial must be a restricted NOLOGIN role';
    END IF;
    EXECUTE format('GRANT grace_financial TO %I', current_user);
    GRANT USAGE ON SCHEMA public TO grace_financial;
    GRANT SELECT ON categories TO grace_financial;
    FOREACH t IN ARRAY ARRAY['accounts','financial_events','postings','receipts',
        'receipt_items','obligations','scheduled_payments','scheduled_payment_settlements']
    LOOP
        EXECUTE format('GRANT SELECT, INSERT, UPDATE ON %I TO grace_financial', t);
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE %I FORCE ROW LEVEL SECURITY', t);
        EXECUTE format('CREATE POLICY owner_access ON %I TO grace_financial USING (owner_id = grace_owner_id()) WITH CHECK (owner_id = grace_owner_id())', t);
    END LOOP;
END $$;
