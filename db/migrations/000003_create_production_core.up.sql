CREATE TABLE assets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    episode_id uuid REFERENCES episodes(id) ON DELETE SET NULL,
    kind text NOT NULL,
    purpose text NOT NULL,
    uri text NOT NULL,
    status text NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_assets_project_id_purpose
    ON assets (project_id, purpose);

CREATE TABLE artifact_edges (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_type text NOT NULL,
    source_id uuid NOT NULL,
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    relationship text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_artifact_edges_project_id_target
    ON artifact_edges (project_id, target_type, target_id);

CREATE TABLE workflow_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    episode_id uuid REFERENCES episodes(id) ON DELETE SET NULL,
    status text NOT NULL,
    graph jsonb NOT NULL DEFAULT '{}'::jsonb,
    input jsonb NOT NULL DEFAULT '{}'::jsonb,
    output jsonb NOT NULL DEFAULT '{}'::jsonb,
    error_message text NOT NULL DEFAULT '',
    started_at timestamptz,
    finished_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_workflow_runs_project_id_created_at
    ON workflow_runs (project_id, created_at DESC);

CREATE TABLE workflow_node_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_run_id uuid NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
    node_key text NOT NULL,
    node_kind text NOT NULL,
    status text NOT NULL,
    input jsonb NOT NULL DEFAULT '{}'::jsonb,
    output jsonb NOT NULL DEFAULT '{}'::jsonb,
    error_message text NOT NULL DEFAULT '',
    started_at timestamptz,
    finished_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (workflow_run_id, node_key)
);

CREATE TABLE generation_jobs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    episode_id uuid REFERENCES episodes(id) ON DELETE SET NULL,
    workflow_run_id uuid REFERENCES workflow_runs(id) ON DELETE SET NULL,
    workflow_node_run_id uuid REFERENCES workflow_node_runs(id) ON DELETE SET NULL,
    request_key text NOT NULL,
    provider text NOT NULL,
    model text NOT NULL,
    task_type text NOT NULL,
    status text NOT NULL,
    prompt text NOT NULL DEFAULT '',
    params jsonb NOT NULL DEFAULT '{}'::jsonb,
    result_asset_id uuid REFERENCES assets(id) ON DELETE SET NULL,
    provider_task_id text NOT NULL DEFAULT '',
    error_message text NOT NULL DEFAULT '',
    estimated_cost_cents integer NOT NULL DEFAULT 0,
    actual_cost_cents integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (request_key)
);

CREATE INDEX idx_generation_jobs_project_id_created_at
    ON generation_jobs (project_id, created_at DESC);

CREATE INDEX idx_generation_jobs_status_created_at
    ON generation_jobs (status, created_at);

CREATE TABLE generation_job_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    generation_job_id uuid NOT NULL REFERENCES generation_jobs(id) ON DELETE CASCADE,
    status text NOT NULL,
    message text NOT NULL DEFAULT '',
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE timelines (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    episode_id uuid NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    status text NOT NULL,
    version integer NOT NULL DEFAULT 1,
    duration_ms integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (episode_id)
);

CREATE TABLE timeline_tracks (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    timeline_id uuid NOT NULL REFERENCES timelines(id) ON DELETE CASCADE,
    kind text NOT NULL,
    name text NOT NULL,
    position integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE timeline_clips (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    timeline_id uuid NOT NULL REFERENCES timelines(id) ON DELETE CASCADE,
    track_id uuid NOT NULL REFERENCES timeline_tracks(id) ON DELETE CASCADE,
    asset_id uuid REFERENCES assets(id) ON DELETE SET NULL,
    kind text NOT NULL,
    start_ms integer NOT NULL,
    duration_ms integer NOT NULL,
    trim_start_ms integer NOT NULL DEFAULT 0,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_timeline_clips_timeline_id_start_ms
    ON timeline_clips (timeline_id, start_ms);

CREATE TABLE exports (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    timeline_id uuid NOT NULL REFERENCES timelines(id) ON DELETE CASCADE,
    status text NOT NULL,
    output_asset_id uuid REFERENCES assets(id) ON DELETE SET NULL,
    format text NOT NULL DEFAULT 'mp4',
    error_message text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
