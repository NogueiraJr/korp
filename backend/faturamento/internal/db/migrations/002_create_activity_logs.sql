CREATE TABLE IF NOT EXISTS activity_logs (
    id         BIGSERIAL PRIMARY KEY,
    event      VARCHAR(60)  NOT NULL,
    entity     VARCHAR(40)  NOT NULL,
    details    JSONB,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_activity_logs_created_at ON activity_logs (created_at DESC);