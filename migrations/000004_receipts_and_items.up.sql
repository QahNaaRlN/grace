CREATE TABLE receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_event_id UUID NOT NULL UNIQUE
        REFERENCES financial_events (id) ON DELETE RESTRICT,
    seller TEXT NOT NULL CHECK (length(trim(seller)) > 0),
    issued_at TIMESTAMPTZ NOT NULL,
    currency CHAR(3) NOT NULL CHECK (currency = upper(currency)),
    total_minor BIGINT NOT NULL CHECK (total_minor > 0),
    fiscal_document_number TEXT NOT NULL DEFAULT '',
    idempotency_key TEXT NOT NULL UNIQUE
        CHECK (length(trim(idempotency_key)) > 0),
    request_hash CHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE receipt_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    receipt_id UUID NOT NULL REFERENCES receipts (id) ON DELETE RESTRICT,
    position INTEGER NOT NULL CHECK (position > 0),
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    quantity_milli BIGINT NOT NULL CHECK (quantity_milli > 0),
    unit_price_minor BIGINT NOT NULL CHECK (unit_price_minor > 0),
    line_total_minor BIGINT NOT NULL CHECK (line_total_minor > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (receipt_id, position)
);

CREATE INDEX receipts_issued_idx ON receipts (issued_at DESC, created_at DESC);
CREATE INDEX receipt_items_receipt_idx ON receipt_items (receipt_id, position);
