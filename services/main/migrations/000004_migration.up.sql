CREATE TABLE teams (
    id         UUID PRIMARY KEY,
    owner_id   UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    city       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_teams_name ON teams (lower(name));

CREATE TABLE team_members (
    player_id UUID PRIMARY KEY REFERENCES players (id) ON DELETE CASCADE,
    team_id   UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE team_invitations (
    id         UUID PRIMARY KEY,
    team_id    UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    player_id  UUID NOT NULL REFERENCES players (id) ON DELETE CASCADE,
    status     TEXT NOT NULL CHECK (status IN ('PENDING', 'ACCEPTED', 'DECLINED', 'CANCELED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_pending_invitation ON team_invitations (team_id, player_id) WHERE status = 'PENDING';
CREATE INDEX idx_invitations_player ON team_invitations (player_id);
