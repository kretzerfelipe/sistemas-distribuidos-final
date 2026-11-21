CREATE TABLE championships (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    slots_total INT NOT NULL DEFAULT 10 CHECK (slots_total > 0),
    slots_taken INT NOT NULL DEFAULT 0 CHECK (slots_taken >= 0),
    status      TEXT NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'IN_PROGRESS', 'FINISHED')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (slots_taken <= slots_total)
);

CREATE TABLE championship_teams (
    championship_id UUID NOT NULL REFERENCES championships (id) ON DELETE CASCADE,
    team_id         UUID NOT NULL REFERENCES teams (id) ON DELETE RESTRICT,
    enrolled_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (championship_id, team_id)
);

CREATE INDEX idx_championship_teams_team ON championship_teams (team_id);
