BEGIN;

CREATE TABLE IF NOT EXISTS subscriptions (
    chat_id      BIGINT PRIMARY KEY,
    frequency    VARCHAR(10) NOT NULL DEFAULT 'daily' CHECK (frequency IN ('hourly', 'daily')),
    next_send_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_due ON subscriptions (next_send_at);

CREATE TABLE IF NOT EXISTS alerts (
    id         CHAR(20) PRIMARY KEY,
    chat_id    BIGINT NOT NULL,
    base       VARCHAR(4) NOT NULL,
    direction  VARCHAR(5) NOT NULL CHECK (direction IN ('above', 'below')),
    threshold  NUMERIC(20,4) NOT NULL,
    triggered  BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_alerts_active ON alerts (triggered) WHERE triggered = false;
CREATE INDEX IF NOT EXISTS idx_alerts_chat ON alerts (chat_id) WHERE triggered = false;

COMMIT;
