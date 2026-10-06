CREATE TABLE payment_instruments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users(id),
    account_id UUID NOT NULL,
    name TEXT NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 160),
    bank TEXT NOT NULL CHECK (length(btrim(bank)) BETWEEN 1 AND 160),
    last4 TEXT NOT NULL CHECK (last4 ~ '^[0-9]{4}$'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (account_id, owner_id) REFERENCES accounts(id, owner_id)
);
CREATE INDEX payment_instruments_account_idx ON payment_instruments(account_id, owner_id);
GRANT SELECT, INSERT ON payment_instruments TO grace_financial;
ALTER TABLE payment_instruments ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_instruments FORCE ROW LEVEL SECURITY;
CREATE POLICY owner_access ON payment_instruments TO grace_financial
    USING (owner_id = grace_owner_id()) WITH CHECK (owner_id = grace_owner_id());
