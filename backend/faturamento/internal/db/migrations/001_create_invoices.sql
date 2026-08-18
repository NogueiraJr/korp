CREATE TABLE IF NOT EXISTS invoice_counters (
    id          INTEGER PRIMARY KEY,
    next_number INTEGER NOT NULL
);

INSERT INTO invoice_counters (id, next_number)
VALUES (1, 0)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS invoices (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    number     INTEGER     NOT NULL UNIQUE,
    status     VARCHAR(10) NOT NULL DEFAULT 'OPEN',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices (status);

CREATE TABLE IF NOT EXISTS invoice_items (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id   UUID        NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    product_code VARCHAR(50) NOT NULL,
    quantity     INTEGER     NOT NULL CHECK (quantity > 0)
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key        VARCHAR(128) PRIMARY KEY,
    response   JSONB       NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);