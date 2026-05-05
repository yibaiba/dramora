package repo

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/yibaiba/dramora/internal/domain"
)

// MemoryBatchSubmissionRepository implements BatchSubmissionRepository using in-memory storage
type MemoryBatchSubmissionRepository struct {
	mu      sync.RWMutex
	batches map[uuid.UUID]*domain.BatchSubmission
}

// NewMemoryBatchSubmissionRepository creates a new in-memory batch submission repository
func NewMemoryBatchSubmissionRepository() *MemoryBatchSubmissionRepository {
	return &MemoryBatchSubmissionRepository{
		batches: make(map[uuid.UUID]*domain.BatchSubmission),
	}
}

// Create inserts a new batch submission
func (r *MemoryBatchSubmissionRepository) Create(ctx context.Context, batch *domain.BatchSubmission) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := batch.Validate(); err != nil {
		return err
	}

	if batch.ID == uuid.Nil {
		batch.ID = uuid.New()
	}

	r.batches[batch.ID] = batch
	return nil
}

// GetByID retrieves a batch submission by ID with organization permission check
func (r *MemoryBatchSubmissionRepository) GetByID(ctx context.Context, id uuid.UUID, orgID uuid.UUID) (*domain.BatchSubmission, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	batch, exists := r.batches[id]
	if !exists || batch.OrganizationID != orgID {
		return nil, domain.ErrNotFound
	}

	return batch, nil
}

// ListByOrganization retrieves all batch submissions for an organization
func (r *MemoryBatchSubmissionRepository) ListByOrganization(ctx context.Context, orgID uuid.UUID, limit int, offset int) ([]*domain.BatchSubmission, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var batches []*domain.BatchSubmission
	for _, batch := range r.batches {
		if batch.OrganizationID == orgID {
			batches = append(batches, batch)
		}
	}

	total := int64(len(batches))

	// Simple pagination
	if offset >= len(batches) {
		return []*domain.BatchSubmission{}, total, nil
	}

	end := offset + limit
	if end > len(batches) {
		end = len(batches)
	}

	return batches[offset:end], total, nil
}

// UpdateStatus updates the status of a batch submission
func (r *MemoryBatchSubmissionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, orgID uuid.UUID, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	batch, exists := r.batches[id]
	if !exists || batch.OrganizationID != orgID {
		return domain.ErrNotFound
	}

	batch.Status = status
	return nil
}

// UpdateProgress updates the progress counters for a batch submission
func (r *MemoryBatchSubmissionRepository) UpdateProgress(ctx context.Context, id uuid.UUID, orgID uuid.UUID, completedCount, failedCount, cancelledCount int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	batch, exists := r.batches[id]
	if !exists || batch.OrganizationID != orgID {
		return domain.ErrNotFound
	}

	batch.CompletedCount = completedCount
	batch.FailedCount = failedCount
	batch.CancelledCount = cancelledCount
	return nil
}

// Delete deletes a batch submission
func (r *MemoryBatchSubmissionRepository) Delete(ctx context.Context, id uuid.UUID, orgID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	batch, exists := r.batches[id]
	if !exists || batch.OrganizationID != orgID {
		return domain.ErrNotFound
	}

	delete(r.batches, id)
	return nil
}
