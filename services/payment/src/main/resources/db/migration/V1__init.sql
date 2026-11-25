CREATE TABLE charges (
  id              UUID PRIMARY KEY,
  subscription_id UUID NOT NULL UNIQUE,
  user_id         UUID NOT NULL,
  amount_cents    INT NOT NULL,
  status          TEXT NOT NULL CHECK (status IN ('PROCESSING', 'APPROVED', 'REJECTED', 'REFUNDED')),
  attempts        INT NOT NULL DEFAULT 0,
  refund_pending  BOOLEAN NOT NULL DEFAULT false,
  failure_reason  TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE outbox (
  id             UUID PRIMARY KEY,
  position       BIGSERIAL UNIQUE,
  correlation_id UUID NOT NULL,
  type           TEXT NOT NULL,
  routing_key    TEXT NOT NULL,
  payload        JSONB NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  published_at   TIMESTAMPTZ
);

CREATE INDEX outbox_unpublished_position ON outbox (position) WHERE published_at IS NULL;

CREATE TABLE processed_messages (
  message_id   UUID PRIMARY KEY,
  processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
