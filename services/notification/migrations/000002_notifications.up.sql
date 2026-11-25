CREATE TABLE notifications (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL,
    type       TEXT NOT NULL,
    message    TEXT NOT NULL,
    read       BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user_created ON notifications (user_id, created_at DESC);
