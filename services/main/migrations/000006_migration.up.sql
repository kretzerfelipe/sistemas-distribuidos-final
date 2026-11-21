CREATE TABLE matches (
    id              UUID PRIMARY KEY,
    championship_id UUID NOT NULL REFERENCES championships (id) ON DELETE RESTRICT,
    home_team_id    UUID NOT NULL REFERENCES teams (id) ON DELETE RESTRICT,
    away_team_id    UUID NOT NULL REFERENCES teams (id) ON DELETE RESTRICT,
    scheduled_at    TIMESTAMPTZ NOT NULL,
    status          TEXT NOT NULL DEFAULT 'SCHEDULED' CHECK (status IN ('SCHEDULED', 'IN_PROGRESS', 'FINISHED')),
    home_score      INT NOT NULL DEFAULT 0 CHECK (home_score >= 0),
    away_score      INT NOT NULL DEFAULT 0 CHECK (away_score >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (home_team_id <> away_team_id)
);

CREATE INDEX idx_matches_championship ON matches (championship_id, scheduled_at);
