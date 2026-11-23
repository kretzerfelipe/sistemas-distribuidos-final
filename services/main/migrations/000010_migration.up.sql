ALTER TABLE teams
    ADD COLUMN photo_key TEXT,
    ADD COLUMN photo_content_type TEXT,
    ADD COLUMN photo_updated_at TIMESTAMPTZ;
