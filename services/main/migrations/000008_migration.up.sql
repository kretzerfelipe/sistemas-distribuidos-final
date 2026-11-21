ALTER TABLE matches ADD COLUMN sequence INT NOT NULL DEFAULT 0;

CREATE TABLE match_events (
    id                    UUID PRIMARY KEY,
    match_id              UUID NOT NULL REFERENCES matches (id) ON DELETE CASCADE,
    sequence              INT NOT NULL,
    type                  TEXT NOT NULL CHECK (type IN (
                              'GOAL', 'OWN_GOAL', 'SHOT', 'SAVE', 'FOUL', 'YELLOW_CARD',
                              'RED_CARD', 'SUBSTITUTION', 'CORNER', 'OFFSIDE')),
    minute                INT NOT NULL CHECK (minute BETWEEN 0 AND 130),
    team_id               UUID NOT NULL REFERENCES teams (id) ON DELETE RESTRICT,
    player_id             UUID REFERENCES players (id) ON DELETE SET NULL,
    player_name           TEXT NOT NULL DEFAULT '',
    secondary_player_id   UUID REFERENCES players (id) ON DELETE SET NULL,
    secondary_player_name TEXT NOT NULL DEFAULT '',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (match_id, sequence)
);

CREATE INDEX idx_match_events_player ON match_events (match_id, player_id);
