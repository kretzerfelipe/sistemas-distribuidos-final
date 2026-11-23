CREATE TABLE subscriptions (
    id           UUID PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users (id),
    status       TEXT NOT NULL CHECK (status IN ('PENDING', 'ACTIVE', 'CANCELED', 'EXPIRED')),
    reason       TEXT,
    charge_id    UUID,
    amount_cents INT NOT NULL CHECK (amount_cents > 0),
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX uq_subscription_in_force ON subscriptions (user_id) WHERE status IN ('PENDING', 'ACTIVE');

CREATE INDEX idx_subscriptions_pending_expiry ON subscriptions (expires_at) WHERE status = 'PENDING';

CREATE TABLE processed_messages (
    message_id   UUID PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
