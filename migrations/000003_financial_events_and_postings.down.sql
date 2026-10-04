DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM financial_events
        WHERE legacy_transaction_id IS NULL
    ) THEN
        RAISE EXCEPTION 'rollback refused: non-legacy financial events exist';
    END IF;
END $$;

CREATE TYPE transaction_direction AS ENUM ('income', 'expense');
CREATE TYPE transaction_kind AS ENUM ('purchase', 'income_received', 'debt_payment', 'refund');
CREATE TYPE transaction_source AS ENUM ('manual', 'bank_import', 'receipt');

ALTER TABLE accounts ADD COLUMN balance_minor BIGINT;

UPDATE accounts a
SET balance_minor = a.opening_balance_minor + COALESCE((
    SELECT SUM(
        CASE p.direction
            WHEN 'credit' THEN p.amount_minor
            ELSE -p.amount_minor
        END
    )
    FROM postings p
    WHERE p.account_id = a.id
), 0);

ALTER TABLE accounts
    ALTER COLUMN balance_minor SET NOT NULL,
    ALTER COLUMN balance_minor SET DEFAULT 0;

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts (id) ON DELETE RESTRICT,
    category_id UUID REFERENCES categories (id) ON DELETE RESTRICT,
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    currency CHAR(3) NOT NULL CHECK (currency = upper(currency)),
    direction transaction_direction NOT NULL,
    kind transaction_kind NOT NULL,
    source transaction_source NOT NULL,
    merchant TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (kind <> 'purchase' OR category_id IS NOT NULL),
    CHECK (
        (kind IN ('purchase', 'debt_payment') AND direction = 'expense')
        OR (kind IN ('income_received', 'refund') AND direction = 'income')
    )
);

INSERT INTO transactions (
    id,
    account_id,
    category_id,
    amount_minor,
    currency,
    direction,
    kind,
    source,
    merchant,
    description,
    occurred_at,
    created_at
)
SELECT
    e.legacy_transaction_id,
    p.account_id,
    e.category_id,
    p.amount_minor,
    p.currency,
    CASE p.direction
        WHEN 'credit' THEN 'income'::transaction_direction
        ELSE 'expense'::transaction_direction
    END,
    e.kind::text::transaction_kind,
    e.source::text::transaction_source,
    e.merchant,
    e.description,
    e.occurred_at,
    e.created_at
FROM financial_events e
JOIN postings p ON p.financial_event_id = e.id;

CREATE INDEX transactions_account_occurred_idx
    ON transactions (account_id, occurred_at DESC);
CREATE INDEX transactions_category_occurred_idx
    ON transactions (category_id, occurred_at DESC);

DROP TABLE postings;
DROP TABLE financial_events;
DROP TYPE posting_direction;
DROP TYPE financial_event_source;
DROP TYPE financial_event_kind;
ALTER TABLE accounts DROP COLUMN opening_balance_minor;
