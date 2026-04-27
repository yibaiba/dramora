CREATE TABLE story_analyses (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    episode_id uuid NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    workflow_run_id uuid REFERENCES workflow_runs(id) ON DELETE SET NULL,
    generation_job_id uuid REFERENCES generation_jobs(id) ON DELETE SET NULL,
    version integer NOT NULL,
    status text NOT NULL,
    summary text NOT NULL DEFAULT '',
    themes jsonb NOT NULL DEFAULT '[]'::jsonb,
    character_seeds jsonb NOT NULL DEFAULT '[]'::jsonb,
    scene_seeds jsonb NOT NULL DEFAULT '[]'::jsonb,
    prop_seeds jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (episode_id, version),
    UNIQUE (generation_job_id)
);

CREATE INDEX idx_story_analyses_episode_id_version
    ON story_analyses (episode_id, version DESC);
