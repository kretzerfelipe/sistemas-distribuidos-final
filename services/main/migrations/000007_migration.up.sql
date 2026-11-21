CREATE TABLE outbox (
    id             UUID PRIMARY KEY,
    position       BIGSERIAL NOT NULL UNIQUE,
    correlation_id UUID NOT NULL,
    type           TEXT NOT NULL,
    routing_key    TEXT NOT NULL,
    payload        JSONB NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at   TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unpublished ON outbox (position) WHERE published_at IS NULL;
