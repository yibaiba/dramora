CREATE TABLE characters (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    episode_id uuid NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    story_analysis_id uuid REFERENCES story_analyses(id) ON DELETE SET NULL,
    code text NOT NULL,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (episode_id, code)
);

CREATE TABLE scenes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    episode_id uuid NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    story_analysis_id uuid REFERENCES story_analyses(id) ON DELETE SET NULL,
    code text NOT NULL,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (episode_id, code)
);

CREATE TABLE props (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    episode_id uuid NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    story_analysis_id uuid REFERENCES story_analyses(id) ON DELETE SET NULL,
    code text NOT NULL,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (episode_id, code)
);

CREATE TABLE storyboard_shots (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    episode_id uuid NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    story_analysis_id uuid REFERENCES story_analyses(id) ON DELETE SET NULL,
    scene_id uuid REFERENCES scenes(id) ON DELETE SET NULL,
    code text NOT NULL,
    title text NOT NULL,
    description text NOT NULL DEFAULT '',
    prompt text NOT NULL DEFAULT '',
    position integer NOT NULL,
    duration_ms integer NOT NULL DEFAULT 3000,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (episode_id, code)
);

CREATE INDEX idx_characters_episode_id_code ON characters (episode_id, code);
CREATE INDEX idx_scenes_episode_id_code ON scenes (episode_id, code);
CREATE INDEX idx_props_episode_id_code ON props (episode_id, code);
CREATE INDEX idx_storyboard_shots_episode_id_position ON storyboard_shots (episode_id, position);
