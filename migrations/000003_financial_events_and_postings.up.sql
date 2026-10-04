CREATE TYPE financial_event_kind AS ENUM (
    'purchase',
    'income_received',
    'internal_transfer',
    'refund',
    'debt_payment'
);
CREATE TYPE financial_event_source AS ENUM (
    'manual',
    'bank_import',
    'receipt'
);
CREATE TYPE posting_direction AS ENUM ('credit', 'debit');

ALTER TABLE accounts ADD COLUMN opening_balance_minor BIGINT;

CREATE TABLE financial_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind financial_event_kind NOT NULL,
    source financial_event_source NOT NULL,
    category_id UUID REFERENCES categories (id) ON DELETE RESTRICT,
    original_event_id UUID REFERENCES financial_events (id) ON DELETE RESTRICT,
    merchant TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL,
    idempotency_key TEXT NOT NULL CHECK (length(trim(idempotency_key)) > 0),
    request_hash CHAR(64) NOT NULL,
    legacy_transaction_id UUID UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (kind, idempotency_key),
    CHECK (kind = 'refund' OR original_event_id IS NULL),
    CHECK (kind <> 'refund' OR original_event_id IS NOT NULL)
);

CREATE TABLE postings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    financial_event_id UUID NOT NULL REFERENCES financial_events (id) ON DELETE RESTRICT,
    account_id UUID NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
    direction posting_direction NOT NULL,
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    currency CHAR(3) NOT NULL CHECK (currency = upper(currency)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX financial_events_occurred_idx
    ON financial_events (occurred_at DESC, created_at DESC);
CREATE INDEX financial_events_original_idx
    ON financial_events (original_event_id)
    WHERE original_event_id IS NOT NULL;
CREATE INDEX postings_event_idx ON postings (financial_event_id);
CREATE INDEX postings_account_idx ON postings (account_id);

INSERT INTO financial_events (
    id,
    kind,
    source,
    category_id,
    merchant,
    description,
    occurred_at,
    idempotency_key,
    request_hash,
    legacy_transaction_id,
    created_at
)
SELECT
    t.id,
    t.kind::text::financial_event_kind,
    t.source::text::financial_event_source,
    t.category_id,
    t.merchant,
    t.description,
    t.occurred_at,
    'legacy:' || t.id::text,
    encode(digest(t.id::text, 'sha256'), 'hex'),
    t.id,
    t.created_at
FROM transactions t;

INSERT INTO postings (
    financial_event_id,
    account_id,
    direction,
    amount_minor,
    currency,
    created_at
)
SELECT
    t.id,
    t.account_id,
    CASE
        WHEN t.direction = 'income' THEN 'credit'::posting_direction
        ELSE 'debit'::posting_direction
    END,
    t.amount_minor,
    t.currency,
    t.created_at
FROM transactions t;

WITH effects AS (
    SELECT
        p.account_id,
        COALESCE(SUM(
            CASE p.direction
                WHEN 'credit' THEN p.amount_minor
                ELSE -p.amount_minor
            END
        ), 0) AS net_effect
    FROM postings p
    GROUP BY p.account_id
)
UPDATE accounts a
SET opening_balance_minor = a.balance_minor - COALESCE(e.net_effect, 0)
FROM effects e
WHERE e.account_id = a.id;

UPDATE accounts
SET opening_balance_minor = balance_minor
WHERE opening_balance_minor IS NULL;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM accounts a
        LEFT JOIN postings p ON p.account_id = a.id
        GROUP BY a.id, a.balance_minor, a.opening_balance_minor
        HAVING a.balance_minor <> a.opening_balance_minor + COALESCE(SUM(
            CASE p.direction
                WHEN 'credit' THEN p.amount_minor
                WHEN 'debit' THEN -p.amount_minor
                ELSE 0
            END
        ), 0)
    ) THEN
        RAISE EXCEPTION 'financial event backfill changed an account balance';
    END IF;
END $$;

ALTER TABLE accounts
    ALTER COLUMN opening_balance_minor SET NOT NULL,
    ALTER COLUMN opening_balance_minor SET DEFAULT 0;

DROP TABLE transactions;
DROP TYPE transaction_source;
DROP TYPE transaction_kind;
DROP TYPE transaction_direction;
ALTER TABLE accounts DROP COLUMN balance_minor;

