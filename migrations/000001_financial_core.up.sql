CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE account_type AS ENUM ('debit', 'credit', 'installment', 'cash');
CREATE TYPE category_kind AS ENUM ('expense', 'income');
CREATE TYPE transaction_direction AS ENUM ('income', 'expense');
CREATE TYPE transaction_kind AS ENUM ('purchase', 'income_received', 'debt_payment', 'refund');
CREATE TYPE transaction_source AS ENUM ('manual', 'bank_import', 'receipt');

CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    type account_type NOT NULL,
    currency CHAR(3) NOT NULL CHECK (currency = upper(currency)),
    balance_minor BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    kind category_kind NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (name, kind)
);

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

CREATE INDEX transactions_account_occurred_idx
    ON transactions (account_id, occurred_at DESC);
CREATE INDEX transactions_category_occurred_idx
    ON transactions (category_id, occurred_at DESC);

