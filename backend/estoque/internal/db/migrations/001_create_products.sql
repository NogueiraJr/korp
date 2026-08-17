CREATE TABLE IF NOT EXISTS products (
    code           VARCHAR(50)  PRIMARY KEY,
    description    VARCHAR(255) NOT NULL,
    stock_quantity INTEGER      NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
