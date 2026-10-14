-- SPEC-0040: accounts and cards are editable, and a balance is confirmed by an
-- append-only reconciliation. A difference on a reconciled account becomes a
-- `balance_adjustment` operation, which moves the balance but is neither an
-- expense nor income in analytics (those queries select kinds explicitly).
ALTER TYPE financial_event_kind ADD VALUE IF NOT EXISTS 'balance_adjustment';

ALTER TABLE accounts ADD COLUMN revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0);

ALTER TABLE payment_instruments ADD COLUMN disabled_at TIMESTAMPTZ;
GRANT UPDATE (name, bank, last4, disabled_at) ON payment_instruments TO grace_financial;

CREATE TABLE account_reconciliations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL DEFAULT grace_owner_id() REFERENCES users (id),
    account_id UUID NOT NULL,
    as_of TIMESTAMPTZ NOT NULL,
    stated_minor BIGINT NOT NULL,
    computed_minor BIGINT NOT NULL,
    difference_minor BIGINT NOT NULL CHECK (difference_minor = stated_minor - computed_minor),
    outcome TEXT NOT NULL CHECK (outcome IN ('matched', 'opening_set', 'adjusted')),
    adjustment_event_id UUID,
    idempotency_key TEXT NOT NULL CHECK (length(idempotency_key) BETWEEN 1 AND 160),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_id, idempotency_key),
    CHECK ((outcome = 'adjusted') = (adjustment_event_id IS NOT NULL)),
    CHECK (outcome <> 'matched' OR difference_minor = 0),
    FOREIGN KEY (account_id, owner_id) REFERENCES accounts (id, owner_id),
    FOREIGN KEY (adjustment_event_id, owner_id) REFERENCES financial_events (id, owner_id)
);
CREATE INDEX account_reconciliations_account_idx ON account_reconciliations (account_id, created_at DESC);
CREATE UNIQUE INDEX account_reconciliations_adjustment_idx ON account_reconciliations (adjustment_event_id)
    WHERE adjustment_event_id IS NOT NULL;
ALTER TABLE account_reconciliations ENABLE ROW LEVEL SECURITY;
ALTER TABLE account_reconciliations FORCE ROW LEVEL SECURITY;
CREATE POLICY owner_access ON account_reconciliations TO grace_financial
    USING (owner_id = grace_owner_id()) WITH CHECK (owner_id = grace_owner_id());
-- Append-only: no UPDATE or DELETE for the application role.
GRANT SELECT, INSERT ON account_reconciliations TO grace_financial;
