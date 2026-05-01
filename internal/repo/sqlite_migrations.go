package repo

var sqliteMigrations = []string{
	// organizations
	`CREATE TABLE IF NOT EXISTS organizations (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,

	// users
	`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		display_name TEXT NOT NULL,
		password_hash TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,

	// organization_members
	`CREATE TABLE IF NOT EXISTS organization_members (
		organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		role TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		PRIMARY KEY (organization_id, user_id)
	)`,

	// projects
	`CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_projects_org_created ON projects (organization_id, created_at)`,

	// episodes
	`CREATE TABLE IF NOT EXISTS episodes (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		number INTEGER NOT NULL,
		title TEXT NOT NULL,
		status TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		UNIQUE (project_id, number)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_episodes_project_number ON episodes (project_id, number)`,

	// assets
	`CREATE TABLE IF NOT EXISTS assets (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		episode_id TEXT REFERENCES episodes(id) ON DELETE SET NULL,
		kind TEXT NOT NULL,
		purpose TEXT NOT NULL,
		uri TEXT NOT NULL,
		status TEXT NOT NULL,
		metadata TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_assets_project_purpose ON assets (project_id, purpose)`,

	// artifact_edges
	`CREATE TABLE IF NOT EXISTS artifact_edges (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		source_type TEXT NOT NULL,
		source_id TEXT NOT NULL,
		target_type TEXT NOT NULL,
		target_id TEXT NOT NULL,
		relationship TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_artifact_edges_target ON artifact_edges (project_id, target_type, target_id)`,

	// workflow_runs
	`CREATE TABLE IF NOT EXISTS workflow_runs (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		episode_id TEXT REFERENCES episodes(id) ON DELETE SET NULL,
		status TEXT NOT NULL,
		graph TEXT NOT NULL DEFAULT '{}',
		input TEXT NOT NULL DEFAULT '{}',
		output TEXT NOT NULL DEFAULT '{}',
		error_message TEXT NOT NULL DEFAULT '',
		started_at TEXT,
		finished_at TEXT,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_workflow_runs_project_created ON workflow_runs (project_id, created_at)`,

	// workflow_node_runs
	`CREATE TABLE IF NOT EXISTS workflow_node_runs (
		id TEXT PRIMARY KEY,
		workflow_run_id TEXT NOT NULL REFERENCES workflow_runs(id) ON DELETE CASCADE,
		node_key TEXT NOT NULL,
		node_kind TEXT NOT NULL,
		status TEXT NOT NULL,
		input TEXT NOT NULL DEFAULT '{}',
		output TEXT NOT NULL DEFAULT '{}',
		error_message TEXT NOT NULL DEFAULT '',
		started_at TEXT,
		finished_at TEXT,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		UNIQUE (workflow_run_id, node_key)
	)`,

	// generation_jobs
	`CREATE TABLE IF NOT EXISTS generation_jobs (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		episode_id TEXT REFERENCES episodes(id) ON DELETE SET NULL,
		workflow_run_id TEXT REFERENCES workflow_runs(id) ON DELETE SET NULL,
		workflow_node_run_id TEXT REFERENCES workflow_node_runs(id) ON DELETE SET NULL,
		request_key TEXT NOT NULL UNIQUE,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		task_type TEXT NOT NULL,
		status TEXT NOT NULL,
		prompt TEXT NOT NULL DEFAULT '',
		params TEXT NOT NULL DEFAULT '{}',
		result_asset_id TEXT REFERENCES assets(id) ON DELETE SET NULL,
		provider_task_id TEXT NOT NULL DEFAULT '',
		error_message TEXT NOT NULL DEFAULT '',
		estimated_cost_cents INTEGER NOT NULL DEFAULT 0,
		actual_cost_cents INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_generation_jobs_project_created ON generation_jobs (project_id, created_at)`,
	`CREATE INDEX IF NOT EXISTS idx_generation_jobs_status_created ON generation_jobs (status, created_at)`,

	// generation_job_events
	`CREATE TABLE IF NOT EXISTS generation_job_events (
		id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
		generation_job_id TEXT NOT NULL REFERENCES generation_jobs(id) ON DELETE CASCADE,
		status TEXT NOT NULL,
		message TEXT NOT NULL DEFAULT '',
		metadata TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,

	// timelines
	`CREATE TABLE IF NOT EXISTS timelines (
		id TEXT PRIMARY KEY,
		episode_id TEXT NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
		status TEXT NOT NULL,
		version INTEGER NOT NULL DEFAULT 1,
		duration_ms INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		UNIQUE (episode_id)
	)`,

	// timeline_tracks
	`CREATE TABLE IF NOT EXISTS timeline_tracks (
		id TEXT PRIMARY KEY,
		timeline_id TEXT NOT NULL REFERENCES timelines(id) ON DELETE CASCADE,
		kind TEXT NOT NULL,
		name TEXT NOT NULL,
		position INTEGER NOT NULL,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,

	// timeline_clips
	`CREATE TABLE IF NOT EXISTS timeline_clips (
		id TEXT PRIMARY KEY,
		timeline_id TEXT NOT NULL REFERENCES timelines(id) ON DELETE CASCADE,
		track_id TEXT NOT NULL REFERENCES timeline_tracks(id) ON DELETE CASCADE,
		asset_id TEXT REFERENCES assets(id) ON DELETE SET NULL,
		kind TEXT NOT NULL,
		start_ms INTEGER NOT NULL,
		duration_ms INTEGER NOT NULL,
		trim_start_ms INTEGER NOT NULL DEFAULT 0,
		metadata TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_timeline_clips_timeline_start ON timeline_clips (timeline_id, start_ms)`,

	// exports
	`CREATE TABLE IF NOT EXISTS exports (
		id TEXT PRIMARY KEY,
		timeline_id TEXT NOT NULL REFERENCES timelines(id) ON DELETE CASCADE,
		status TEXT NOT NULL,
		output_asset_id TEXT REFERENCES assets(id) ON DELETE SET NULL,
		format TEXT NOT NULL DEFAULT 'mp4',
		error_message TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_exports_status_created ON exports (status, created_at)`,

	// story_sources (must precede story_analyses due to FK)
	`CREATE TABLE IF NOT EXISTS story_sources (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		episode_id TEXT NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
		source_type TEXT NOT NULL DEFAULT 'novel',
		title TEXT NOT NULL DEFAULT '',
		content_text TEXT NOT NULL DEFAULT '',
		language TEXT NOT NULL DEFAULT 'zh-CN',
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_story_sources_episode_created ON story_sources (episode_id, created_at)`,

	// story_analyses
	`CREATE TABLE IF NOT EXISTS story_analyses (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		episode_id TEXT NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
		story_source_id TEXT REFERENCES story_sources(id) ON DELETE SET NULL,
		workflow_run_id TEXT REFERENCES workflow_runs(id) ON DELETE SET NULL,
		generation_job_id TEXT REFERENCES generation_jobs(id) ON DELETE SET NULL,
		version INTEGER NOT NULL,
		status TEXT NOT NULL,
		summary TEXT NOT NULL DEFAULT '',
		themes TEXT NOT NULL DEFAULT '[]',
		character_seeds TEXT NOT NULL DEFAULT '[]',
		scene_seeds TEXT NOT NULL DEFAULT '[]',
		prop_seeds TEXT NOT NULL DEFAULT '[]',
		outline TEXT NOT NULL DEFAULT '[]',
		agent_outputs TEXT NOT NULL DEFAULT '[]',
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		UNIQUE (episode_id, version),
		UNIQUE (generation_job_id)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_story_analyses_episode_version ON story_analyses (episode_id, version)`,

	// characters
	`CREATE TABLE IF NOT EXISTS characters (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		episode_id TEXT NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
		story_analysis_id TEXT REFERENCES story_analyses(id) ON DELETE SET NULL,
		code TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		character_bible TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		UNIQUE (episode_id, code)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_characters_episode_code ON characters (episode_id, code)`,
	`ALTER TABLE characters ADD COLUMN character_bible TEXT NOT NULL DEFAULT '{}'`,

	// scenes
	`CREATE TABLE IF NOT EXISTS scenes (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		episode_id TEXT NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
		story_analysis_id TEXT REFERENCES story_analyses(id) ON DELETE SET NULL,
		code TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		UNIQUE (episode_id, code)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_scenes_episode_code ON scenes (episode_id, code)`,

	// props
	`CREATE TABLE IF NOT EXISTS props (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		episode_id TEXT NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
		story_analysis_id TEXT REFERENCES story_analyses(id) ON DELETE SET NULL,
		code TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		UNIQUE (episode_id, code)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_props_episode_code ON props (episode_id, code)`,

	// storyboard_shots
	`CREATE TABLE IF NOT EXISTS storyboard_shots (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		episode_id TEXT NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
		story_analysis_id TEXT REFERENCES story_analyses(id) ON DELETE SET NULL,
		scene_id TEXT REFERENCES scenes(id) ON DELETE SET NULL,
		code TEXT NOT NULL,
		title TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		prompt TEXT NOT NULL DEFAULT '',
		position INTEGER NOT NULL,
		duration_ms INTEGER NOT NULL DEFAULT 3000,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		UNIQUE (episode_id, code)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_storyboard_shots_episode_position ON storyboard_shots (episode_id, position)`,

	// shot_prompt_packs
	`CREATE TABLE IF NOT EXISTS shot_prompt_packs (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		episode_id TEXT NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
		shot_id TEXT NOT NULL REFERENCES storyboard_shots(id) ON DELETE CASCADE,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		preset TEXT NOT NULL,
		task_type TEXT NOT NULL,
		direct_prompt TEXT NOT NULL,
		negative_prompt TEXT NOT NULL DEFAULT '',
		time_slices TEXT NOT NULL DEFAULT '[]',
		reference_bindings TEXT NOT NULL DEFAULT '[]',
		params TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		UNIQUE (shot_id, preset)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_shot_prompt_packs_episode ON shot_prompt_packs (episode_id, updated_at)`,

	// approval_gates
	`CREATE TABLE IF NOT EXISTS approval_gates (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		episode_id TEXT NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
		workflow_run_id TEXT REFERENCES workflow_runs(id) ON DELETE SET NULL,
		gate_type TEXT NOT NULL,
		subject_type TEXT NOT NULL,
		subject_id TEXT NOT NULL,
		status TEXT NOT NULL,
		reviewed_by TEXT NOT NULL DEFAULT '',
		review_note TEXT NOT NULL DEFAULT '',
		reviewed_at TEXT,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		UNIQUE (episode_id, gate_type, subject_type, subject_id)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_approval_gates_episode_status ON approval_gates (episode_id, status, created_at)`,

	// Phase 1: provider_configs
	`CREATE TABLE IF NOT EXISTS provider_configs (
		id TEXT PRIMARY KEY,
		capability TEXT NOT NULL UNIQUE,
		provider_type TEXT NOT NULL DEFAULT 'openai',
		base_url TEXT NOT NULL,
		api_key TEXT NOT NULL,
		model TEXT NOT NULL,
		credits_per_unit INTEGER NOT NULL DEFAULT 0,
		credit_unit TEXT NOT NULL DEFAULT 'per_call',
		timeout_ms INTEGER DEFAULT 120000,
		max_retries INTEGER DEFAULT 3,
		is_enabled INTEGER DEFAULT 1,
		updated_at TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_by TEXT
	)`,
	`ALTER TABLE provider_configs ADD COLUMN provider_type TEXT NOT NULL DEFAULT 'openai'`,

	// Phase 2: organization_invitations
	`CREATE TABLE IF NOT EXISTS organization_invitations (
		id TEXT PRIMARY KEY,
		organization_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
		email TEXT NOT NULL,
		role TEXT NOT NULL,
		token TEXT NOT NULL UNIQUE,
		invited_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		expires_at TEXT NOT NULL,
		accepted_at TEXT,
		accepted_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_invitations_org_status ON organization_invitations (organization_id, status)`,

	// Phase 2: worker_metric_state — restart-survivable counters for worker observability.
	`CREATE TABLE IF NOT EXISTS worker_metric_state (
		metric_kind TEXT PRIMARY KEY,
		counter INTEGER NOT NULL DEFAULT 0,
		last_reason TEXT,
		last_at TEXT,
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,

	// 弱化历史 default organization 种子（仅在无引用时清理）。
	`DELETE FROM organizations
		WHERE id = '00000000-0000-0000-0000-000000000001'
		  AND NOT EXISTS (SELECT 1 FROM organization_members WHERE organization_id = '00000000-0000-0000-0000-000000000001')
		  AND NOT EXISTS (SELECT 1 FROM projects WHERE organization_id = '00000000-0000-0000-0000-000000000001')
		  AND NOT EXISTS (SELECT 1 FROM organization_invitations WHERE organization_id = '00000000-0000-0000-0000-000000000001')`,

	// PR8: refresh token 表 — 短 access (1h) + 长 refresh (30d, 旋转)。
	`CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		organization_id TEXT NOT NULL,
		role TEXT NOT NULL,
		token_hash TEXT NOT NULL UNIQUE,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		expires_at TEXT NOT NULL,
		revoked_at TEXT,
		replaced_by_id TEXT
	)`,
	`CREATE INDEX IF NOT EXISTS idx_auth_refresh_tokens_user_active
		ON auth_refresh_tokens (user_id, revoked_at)`,

	// PR8: 邀请审计事件 — 记录 created/accepted/revoked/resent 关键动作。
	`CREATE TABLE IF NOT EXISTS organization_invitation_events (
		id TEXT PRIMARY KEY,
		organization_id TEXT NOT NULL,
		invitation_id TEXT NOT NULL,
		action TEXT NOT NULL,
		actor_user_id TEXT,
		actor_email TEXT,
		email TEXT NOT NULL,
		role TEXT NOT NULL,
		note TEXT,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_invitation_events_org_created
		ON organization_invitation_events (organization_id, created_at DESC)`,

	// PR8: provider 审计事件 — 记录 admin 对 provider config 的 save/test 关键动作。
	`CREATE TABLE IF NOT EXISTS provider_audit_events (
		id TEXT PRIMARY KEY,
		organization_id TEXT NOT NULL,
		action TEXT NOT NULL,
		actor_user_id TEXT,
		actor_email TEXT,
		capability TEXT NOT NULL,
		provider_type TEXT NOT NULL,
		model TEXT,
		success INTEGER NOT NULL,
		message TEXT,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,
	`CREATE INDEX IF NOT EXISTS idx_provider_audit_events_org_created
		ON provider_audit_events (organization_id, created_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_provider_audit_events_capability
		ON provider_audit_events (organization_id, capability)`,

	// PR2: llm_telemetry_aggregate — restart-survivable per-vendor / per-capability counters.
	`CREATE TABLE IF NOT EXISTS llm_telemetry_aggregate (
		scope TEXT NOT NULL,
		key TEXT NOT NULL,
		counter INTEGER NOT NULL DEFAULT 0,
		error_counter INTEGER NOT NULL DEFAULT 0,
		total_duration_ms INTEGER NOT NULL DEFAULT 0,
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		PRIMARY KEY (scope, key)
	)`,

	// PR2 window: daily buckets for rolling N-day telemetry views.
	`CREATE TABLE IF NOT EXISTS llm_telemetry_daily (
		scope TEXT NOT NULL,
		key TEXT NOT NULL,
		day_utc TEXT NOT NULL,
		counter INTEGER NOT NULL DEFAULT 0,
		error_counter INTEGER NOT NULL DEFAULT 0,
		total_duration_ms INTEGER NOT NULL DEFAULT 0,
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
		PRIMARY KEY (scope, key, day_utc)
	)`,
	`CREATE INDEX IF NOT EXISTS llm_telemetry_daily_day_idx ON llm_telemetry_daily (day_utc)`,

	// PR13 wallet/credits.
	`CREATE TABLE IF NOT EXISTS wallets (
		organization_id TEXT PRIMARY KEY,
		balance INTEGER NOT NULL DEFAULT 0,
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,
	`CREATE TABLE IF NOT EXISTS wallet_transactions (
		id TEXT PRIMARY KEY,
		organization_id TEXT NOT NULL,
		kind TEXT NOT NULL,
		direction INTEGER NOT NULL DEFAULT 1,
		amount INTEGER NOT NULL,
		reason TEXT,
		ref_type TEXT,
		ref_id TEXT,
		balance_after INTEGER NOT NULL,
		actor_user_id TEXT,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
	)`,
	`CREATE INDEX IF NOT EXISTS wallet_transactions_org_created_idx
		ON wallet_transactions (organization_id, created_at DESC)`,
	`CREATE INDEX IF NOT EXISTS wallet_transactions_ref_idx
		ON wallet_transactions (organization_id, ref_type, ref_id)`,
}
