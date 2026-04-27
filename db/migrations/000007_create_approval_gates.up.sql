CREATE TABLE approval_gates (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    episode_id uuid NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    workflow_run_id uuid REFERENCES workflow_runs(id) ON DELETE SET NULL,
    gate_type text NOT NULL,
    subject_type text NOT NULL,
    subject_id uuid NOT NULL,
    status text NOT NULL,
    reviewed_by text NOT NULL DEFAULT '',
    review_note text NOT NULL DEFAULT '',
    reviewed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (episode_id, gate_type, subject_type, subject_id)
);

CREATE INDEX idx_approval_gates_episode_id_status
    ON approval_gates (episode_id, status, created_at);

