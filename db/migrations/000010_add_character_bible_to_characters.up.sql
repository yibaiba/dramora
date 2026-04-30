ALTER TABLE characters
    ADD COLUMN character_bible jsonb NOT NULL DEFAULT '{}'::jsonb;
