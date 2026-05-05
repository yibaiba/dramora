package repo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yibaiba/dramora/internal/domain"
)

// BatchSubmissionRepository defines operations for batch submissions
type BatchSubmissionRepository interface {
	Create(ctx context.Context, batch *domain.BatchSubmission) error
	GetByID(ctx context.Context, id uuid.UUID, orgID uuid.UUID) (*domain.BatchSubmission, error)
	ListByOrganization(ctx context.Context, orgID uuid.UUID, limit int, offset int) ([]*domain.BatchSubmission, int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, orgID uuid.UUID, status string) error
	UpdateProgress(ctx context.Context, id uuid.UUID, orgID uuid.UUID, completedCount, failedCount, cancelledCount int) error
	Delete(ctx context.Context, id uuid.UUID, orgID uuid.UUID) error
}

// PostgresBatchSubmissionRepository implements BatchSubmissionRepository using PostgreSQL
type PostgresBatchSubmissionRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresBatchSubmissionRepository creates a new batch submission repository
func NewPostgresBatchSubmissionRepository(pool *pgxpool.Pool) *PostgresBatchSubmissionRepository {
	return &PostgresBatchSubmissionRepository{pool: pool}
}

// Create inserts a new batch submission
func (r *PostgresBatchSubmissionRepository) Create(ctx context.Context, batch *domain.BatchSubmission) error {
	if err := batch.Validate(); err != nil {
		return err
	}

	if batch.ID == uuid.Nil {
		batch.ID = uuid.New()
	}

	now := time.Now()
	batch.CreatedAt = now
	batch.UpdatedAt = now

	query := `
		INSERT INTO batch_submissions 
		(id, organization_id, created_by_user_id, name, description, total_count, 
		 completed_count, failed_count, cancelled_count, status, concurrency_limit, 
		 retry_limit, created_at, updated_at, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	_, err := r.pool.Exec(ctx, query,
		batch.ID,
		batch.OrganizationID,
		batch.CreatedByUserID,
		batch.Name,
		batch.Description,
		batch.TotalCount,
		batch.CompletedCount,
		batch.FailedCount,
		batch.CancelledCount,
		batch.Status,
		batch.ConcurrencyLimit,
		batch.RetryLimit,
		batch.CreatedAt,
		batch.UpdatedAt,
		batch.StartedAt,
		batch.CompletedAt,
	)

	return err
}

// GetByID retrieves a batch submission by ID with organization permission check
func (r *PostgresBatchSubmissionRepository) GetByID(ctx context.Context, id uuid.UUID, orgID uuid.UUID) (*domain.BatchSubmission, error) {
	var batch domain.BatchSubmission

	query := `
		SELECT id, organization_id, created_by_user_id, name, description, total_count,
		       completed_count, failed_count, cancelled_count, status, concurrency_limit,
		       retry_limit, created_at, updated_at, started_at, completed_at
		FROM batch_submissions
		WHERE id = $1 AND organization_id = $2
	`

	err := r.pool.QueryRow(ctx, query, id, orgID).Scan(
		&batch.ID,
		&batch.OrganizationID,
		&batch.CreatedByUserID,
		&batch.Name,
		&batch.Description,
		&batch.TotalCount,
		&batch.CompletedCount,
		&batch.FailedCount,
		&batch.CancelledCount,
		&batch.Status,
		&batch.ConcurrencyLimit,
		&batch.RetryLimit,
		&batch.CreatedAt,
		&batch.UpdatedAt,
		&batch.StartedAt,
		&batch.CompletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &batch, nil
}

// ListByOrganization retrieves all batch submissions for an organization
func (r *PostgresBatchSubmissionRepository) ListByOrganization(ctx context.Context, orgID uuid.UUID, limit int, offset int) ([]*domain.BatchSubmission, int64, error) {
	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM batch_submissions WHERE organization_id = $1`
	err := r.pool.QueryRow(ctx, countQuery, orgID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated results
	query := `
		SELECT id, organization_id, created_by_user_id, name, description, total_count,
		       completed_count, failed_count, cancelled_count, status, concurrency_limit,
		       retry_limit, created_at, updated_at, started_at, completed_at
		FROM batch_submissions
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	batches := make([]*domain.BatchSubmission, 0, limit)
	for rows.Next() {
		var batch domain.BatchSubmission
		err := rows.Scan(
			&batch.ID,
			&batch.OrganizationID,
			&batch.CreatedByUserID,
			&batch.Name,
			&batch.Description,
			&batch.TotalCount,
			&batch.CompletedCount,
			&batch.FailedCount,
			&batch.CancelledCount,
			&batch.Status,
			&batch.ConcurrencyLimit,
			&batch.RetryLimit,
			&batch.CreatedAt,
			&batch.UpdatedAt,
			&batch.StartedAt,
			&batch.CompletedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		batches = append(batches, &batch)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	return batches, total, nil
}

// UpdateStatus updates the status of a batch submission
func (r *PostgresBatchSubmissionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, orgID uuid.UUID, status string) error {
	now := time.Now()
	var startedAt, completedAt interface{}

	// Set timestamps based on status
	if status == domain.BatchStatusProcessing {
		startedAt = now
	}
	if status == domain.BatchStatusCompleted || status == domain.BatchStatusFailed || status == domain.BatchStatusCancelled {
		completedAt = now
	}

	query := `
		UPDATE batch_submissions
		SET status = $1, updated_at = $2, started_at = COALESCE(started_at, $3), completed_at = COALESCE(completed_at, $4)
		WHERE id = $5 AND organization_id = $6
	`

	result, err := r.pool.Exec(ctx, query, status, now, startedAt, completedAt, id, orgID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// UpdateProgress updates the progress counters for a batch submission
func (r *PostgresBatchSubmissionRepository) UpdateProgress(ctx context.Context, id uuid.UUID, orgID uuid.UUID, completedCount, failedCount, cancelledCount int) error {
	query := `
		UPDATE batch_submissions
		SET completed_count = $1, failed_count = $2, cancelled_count = $3, updated_at = $4
		WHERE id = $5 AND organization_id = $6
	`

	result, err := r.pool.Exec(ctx, query, completedCount, failedCount, cancelledCount, time.Now(), id, orgID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// Delete deletes a batch submission (soft delete via status change recommended in real scenarios)
func (r *PostgresBatchSubmissionRepository) Delete(ctx context.Context, id uuid.UUID, orgID uuid.UUID) error {
	query := `DELETE FROM batch_submissions WHERE id = $1 AND organization_id = $2`

	result, err := r.pool.Exec(ctx, query, id, orgID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}
