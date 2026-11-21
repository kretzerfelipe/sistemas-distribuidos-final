CREATE TABLE players (
    id           UUID PRIMARY KEY,
    user_id      UUID NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    position     TEXT NOT NULL CHECK (position IN ('GOALKEEPER', 'DEFENDER', 'MIDFIELDER', 'FORWARD')),
    shirt_number INT NOT NULL CHECK (shirt_number BETWEEN 1 AND 99),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
