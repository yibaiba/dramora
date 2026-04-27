CREATE TABLE shot_prompt_packs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    episode_id uuid NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    shot_id uuid NOT NULL REFERENCES storyboard_shots(id) ON DELETE CASCADE,
    provider text NOT NULL,
    model text NOT NULL,
    preset text NOT NULL,
    task_type text NOT NULL,
    direct_prompt text NOT NULL,
    negative_prompt text NOT NULL DEFAULT '',
    time_slices jsonb NOT NULL DEFAULT '[]'::jsonb,
    reference_bindings jsonb NOT NULL DEFAULT '[]'::jsonb,
    params jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (shot_id, preset)
);

CREATE INDEX idx_shot_prompt_packs_episode_id ON shot_prompt_packs (episode_id, updated_at DESC);
