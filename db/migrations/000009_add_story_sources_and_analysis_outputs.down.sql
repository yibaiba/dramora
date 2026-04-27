ALTER TABLE story_analyses
    DROP COLUMN IF EXISTS agent_outputs,
    DROP COLUMN IF EXISTS outline,
    DROP COLUMN IF EXISTS story_source_id;

DROP TABLE IF EXISTS story_sources;
