CREATE TABLE story_sources (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    episode_id uuid NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    source_type text NOT NULL DEFAULT 'novel',
    title text NOT NULL DEFAULT '',
    content_text text NOT NULL DEFAULT '',
    language text NOT NULL DEFAULT 'zh-CN',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_story_sources_episode_id_created_at
    ON story_sources (episode_id, created_at DESC);

ALTER TABLE story_analyses
    ADD COLUMN story_source_id uuid REFERENCES story_sources(id) ON DELETE SET NULL,
    ADD COLUMN outline jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN agent_outputs jsonb NOT NULL DEFAULT '[]'::jsonb;
