package repo

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yibaiba/dramora/internal/domain"
)

// ShortVideoTemplateRepository defines operations for short video templates
type ShortVideoTemplateRepository interface {
	Create(ctx context.Context, template *domain.ShortVideoTemplate) error
	GetByID(ctx context.Context, id uuid.UUID, orgID uuid.UUID) (*domain.ShortVideoTemplate, error)
	ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]*domain.ShortVideoTemplate, error)
	Update(ctx context.Context, template *domain.ShortVideoTemplate) error
	Delete(ctx context.Context, id uuid.UUID, orgID uuid.UUID) error
}

// ShortVideoRepository defines operations for short videos
type ShortVideoRepository interface {
	Create(ctx context.Context, video *domain.ShortVideo) error
	GetByID(ctx context.Context, id uuid.UUID, orgID uuid.UUID) (*domain.ShortVideo, error)
	ListByOrganization(ctx context.Context, orgID uuid.UUID, limit int, offset int) ([]*domain.ShortVideo, int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, orgID uuid.UUID, status string, errorMsg *string, result *domain.ShortVideoResult) error
	Delete(ctx context.Context, id uuid.UUID, orgID uuid.UUID) error
}

// PostgresShortVideoTemplateRepository implements ShortVideoTemplateRepository using PostgreSQL
type PostgresShortVideoTemplateRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresShortVideoTemplateRepository creates a new template repository
func NewPostgresShortVideoTemplateRepository(pool *pgxpool.Pool) *PostgresShortVideoTemplateRepository {
	return &PostgresShortVideoTemplateRepository{pool: pool}
}

// Create inserts a new template
func (r *PostgresShortVideoTemplateRepository) Create(ctx context.Context, template *domain.ShortVideoTemplate) error {
	if err := template.Validate(); err != nil {
		return err
	}

	if template.ID == uuid.Nil {
		template.ID = uuid.New()
	}

	query := `
		INSERT INTO short_video_templates (id, organization_id, name, description, category, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	err := r.pool.QueryRow(ctx, query,
		template.ID,
		template.OrganizationID,
		template.Name,
		template.Description,
		template.Category,
		template.Config,
		template.CreatedAt,
		template.UpdatedAt,
	).Scan()

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil // INSERT returns no rows, so ErrNoRows is expected and not an error
		}
		return err
	}

	return nil
}

// GetByID retrieves a template by ID with organization permission check
func (r *PostgresShortVideoTemplateRepository) GetByID(ctx context.Context, id uuid.UUID, orgID uuid.UUID) (*domain.ShortVideoTemplate, error) {
	var template domain.ShortVideoTemplate

	query := `
		SELECT id, organization_id, name, description, category, config, created_at, updated_at
		FROM short_video_templates
		WHERE id = $1 AND organization_id = $2
	`

	err := r.pool.QueryRow(ctx, query, id, orgID).Scan(
		&template.ID,
		&template.OrganizationID,
		&template.Name,
		&template.Description,
		&template.Category,
		&template.Config,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &template, nil
}

// ListByOrganization retrieves all templates for an organization
func (r *PostgresShortVideoTemplateRepository) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]*domain.ShortVideoTemplate, error) {
	query := `
		SELECT id, organization_id, name, description, category, config, created_at, updated_at
		FROM short_video_templates
		WHERE organization_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*domain.ShortVideoTemplate
	for rows.Next() {
		var template domain.ShortVideoTemplate
		if err := rows.Scan(
			&template.ID,
			&template.OrganizationID,
			&template.Name,
			&template.Description,
			&template.Category,
			&template.Config,
			&template.CreatedAt,
			&template.UpdatedAt,
		); err != nil {
			return nil, err
		}
		templates = append(templates, &template)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return templates, nil
}

// Update updates an existing template
func (r *PostgresShortVideoTemplateRepository) Update(ctx context.Context, template *domain.ShortVideoTemplate) error {
	if err := template.Validate(); err != nil {
		return err
	}

	query := `
		UPDATE short_video_templates
		SET name = $1, description = $2, category = $3, config = $4, updated_at = $5
		WHERE id = $6 AND organization_id = $7
	`

	commandTag, err := r.pool.Exec(ctx, query,
		template.Name,
		template.Description,
		template.Category,
		template.Config,
		template.UpdatedAt,
		template.ID,
		template.OrganizationID,
	)

	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// Delete removes a template
func (r *PostgresShortVideoTemplateRepository) Delete(ctx context.Context, id uuid.UUID, orgID uuid.UUID) error {
	query := `
		DELETE FROM short_video_templates
		WHERE id = $1 AND organization_id = $2
	`

	commandTag, err := r.pool.Exec(ctx, query, id, orgID)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// PostgresShortVideoRepository implements ShortVideoRepository using PostgreSQL
type PostgresShortVideoRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresShortVideoRepository creates a new short video repository
func NewPostgresShortVideoRepository(pool *pgxpool.Pool) *PostgresShortVideoRepository {
	return &PostgresShortVideoRepository{pool: pool}
}

// Create inserts a new short video
func (r *PostgresShortVideoRepository) Create(ctx context.Context, video *domain.ShortVideo) error {
	if err := video.Validate(); err != nil {
		return err
	}

	if video.ID == uuid.Nil {
		video.ID = uuid.New()
	}

	// Use generationStatus if available, otherwise use Status
	generationStatus := video.GenerationStatus
	if generationStatus == "" {
		generationStatus = video.Status
	}

	query := `
		INSERT INTO short_videos (id, organization_id, template_id, parameters, heygen_avatar_id, heygen_video_id, generation_status, status, error_message, batch_id, retry_count, created_by_batch, created_at, updated_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	err := r.pool.QueryRow(ctx, query,
		video.ID,
		video.OrganizationID,
		video.TemplateID,
		video.Parameters,
		video.HeyGenAvatarID,
		video.HeyGenVideoID,
		generationStatus,
		video.Status,
		video.ErrorMessage,
		video.BatchID,
		video.RetryCount,
		video.CreatedByBatch,
		video.CreatedAt,
		video.UpdatedAt,
		video.Version,
	).Scan()

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil // INSERT returns no rows, so ErrNoRows is expected
		}
		return err
	}

	return nil
}

// GetByID retrieves a short video by ID with organization permission check
func (r *PostgresShortVideoRepository) GetByID(ctx context.Context, id uuid.UUID, orgID uuid.UUID) (*domain.ShortVideo, error) {
	var video domain.ShortVideo
	var result *string

	query := `
		SELECT id, organization_id, template_id, parameters, heygen_avatar_id, heygen_video_id, generation_status, status, error_message, result, batch_id, retry_count, created_by_batch, created_at, updated_at, version
		FROM short_videos
		WHERE id = $1 AND organization_id = $2
	`

	err := r.pool.QueryRow(ctx, query, id, orgID).Scan(
		&video.ID,
		&video.OrganizationID,
		&video.TemplateID,
		&video.Parameters,
		&video.HeyGenAvatarID,
		&video.HeyGenVideoID,
		&video.GenerationStatus,
		&video.Status,
		&video.ErrorMessage,
		&result,
		&video.BatchID,
		&video.RetryCount,
		&video.CreatedByBatch,
		&video.CreatedAt,
		&video.UpdatedAt,
		&video.Version,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	// Parse result JSON if present
	if result != nil {
		var res domain.ShortVideoResult
		if err := json.Unmarshal([]byte(*result), &res); err != nil {
			return nil, err
		}
		video.Result = &res
	}

	return &video, nil
}

// ListByOrganization retrieves all short videos for an organization with pagination
func (r *PostgresShortVideoRepository) ListByOrganization(ctx context.Context, orgID uuid.UUID, limit int, offset int) ([]*domain.ShortVideo, int64, error) {
	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM short_videos WHERE organization_id = $1`
	err := r.pool.QueryRow(ctx, countQuery, orgID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query := `
		SELECT id, organization_id, template_id, parameters, heygen_avatar_id, heygen_video_id, generation_status, status, error_message, result, batch_id, retry_count, created_by_batch, created_at, updated_at, version
		FROM short_videos
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var videos []*domain.ShortVideo
	for rows.Next() {
		var video domain.ShortVideo
		var result *string

		if err := rows.Scan(
			&video.ID,
			&video.OrganizationID,
			&video.TemplateID,
			&video.Parameters,
			&video.HeyGenAvatarID,
			&video.HeyGenVideoID,
			&video.GenerationStatus,
			&video.Status,
			&video.ErrorMessage,
			&result,
			&video.BatchID,
			&video.RetryCount,
			&video.CreatedByBatch,
			&video.CreatedAt,
			&video.UpdatedAt,
			&video.Version,
		); err != nil {
			return nil, 0, err
		}

		// Parse result JSON if present
		if result != nil {
			var res domain.ShortVideoResult
			if err := json.Unmarshal([]byte(*result), &res); err != nil {
				return nil, 0, err
			}
			video.Result = &res
		}

		videos = append(videos, &video)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return videos, total, nil
}

// UpdateStatus updates the status of a short video
func (r *PostgresShortVideoRepository) UpdateStatus(ctx context.Context, id uuid.UUID, orgID uuid.UUID, status string, errorMsg *string, result *domain.ShortVideoResult) error {
	// Validate status transition
	var currentStatus string
	query := `SELECT status FROM short_videos WHERE id = $1 AND organization_id = $2`
	err := r.pool.QueryRow(ctx, query, id, orgID).Scan(&currentStatus)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.ErrNotFound
		}
		return err
	}

	if !domain.CanTransitionTo(currentStatus, status) {
		return errors.New("invalid status transition")
	}

	// Prepare result JSON if provided
	var resultJSON *[]byte
	if result != nil {
		b, err := json.Marshal(result)
		if err != nil {
			return err
		}
		resultJSON = &b
	}

	// Update status
	updateQuery := `
		UPDATE short_videos
		SET status = $1, error_message = $2, result = $3, updated_at = NOW(), version = version + 1
		WHERE id = $4 AND organization_id = $5
	`

	commandTag, err := r.pool.Exec(ctx, updateQuery,
		status,
		errorMsg,
		resultJSON,
		id,
		orgID,
	)

	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// Delete removes a short video
func (r *PostgresShortVideoRepository) Delete(ctx context.Context, id uuid.UUID, orgID uuid.UUID) error {
	query := `
		DELETE FROM short_videos
		WHERE id = $1 AND organization_id = $2
	`

	commandTag, err := r.pool.Exec(ctx, query, id, orgID)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}
