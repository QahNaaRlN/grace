CREATE TABLE obligations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    kind TEXT NOT NULL CHECK (kind IN ('loan', 'installment', 'other_fixed')),
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    total_minor BIGINT NOT NULL CHECK (total_minor > 0),
    idempotency_key TEXT NOT NULL UNIQUE
        CHECK (length(trim(idempotency_key)) > 0),
    request_hash CHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE scheduled_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    obligation_id UUID NOT NULL REFERENCES obligations (id) ON DELETE RESTRICT,
    position INTEGER NOT NULL CHECK (position > 0),
    due_on DATE NOT NULL,
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (obligation_id, position),
    UNIQUE (obligation_id, due_on)
);

CREATE TABLE scheduled_payment_settlements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheduled_payment_id UUID NOT NULL
        REFERENCES scheduled_payments (id) ON DELETE RESTRICT,
    financial_event_id UUID NOT NULL UNIQUE
        REFERENCES financial_events (id) ON DELETE RESTRICT,
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    idempotency_key TEXT NOT NULL UNIQUE
        CHECK (length(trim(idempotency_key)) > 0),
    request_hash CHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX obligations_created_idx ON obligations (created_at DESC);
CREATE INDEX scheduled_payments_due_idx ON scheduled_payments (due_on, obligation_id);
CREATE INDEX scheduled_payments_obligation_idx
    ON scheduled_payments (obligation_id, position);
CREATE INDEX scheduled_payment_settlements_payment_idx
    ON scheduled_payment_settlements (scheduled_payment_id, created_at);
