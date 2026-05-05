package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

// BatchSubmissionService manages batch video generation operations
type BatchSubmissionService struct {
	batchRepo     repo.BatchSubmissionRepository
	videoRepo     repo.ShortVideoRepository
	productionSvc *ProductionService
	queueMutex    sync.Mutex
	activeQueues  map[uuid.UUID]*BatchQueue
}

// BatchQueue represents an active batch generation queue
type BatchQueue struct {
	BatchID        uuid.UUID
	OrganizationID uuid.UUID
	Concurrency    int
	RetryLimit     int
	Workers        int
	TotalCount     int
	Active         bool
}

// NewBatchSubmissionService creates a new batch submission service
func NewBatchSubmissionService(
	batchRepo repo.BatchSubmissionRepository,
	videoRepo repo.ShortVideoRepository,
	productionSvc *ProductionService,
) *BatchSubmissionService {
	return &BatchSubmissionService{
		batchRepo:     batchRepo,
		videoRepo:     videoRepo,
		productionSvc: productionSvc,
		activeQueues:  make(map[uuid.UUID]*BatchQueue),
	}
}

// CreateBatchSubmission creates a new batch submission and queues it for processing
func (s *BatchSubmissionService) CreateBatchSubmission(ctx context.Context, batch *domain.BatchSubmission) error {
	// Validate batch
	if err := batch.Validate(); err != nil {
		return err
	}

	// Generate ID if not provided
	if batch.ID == uuid.Nil {
		batch.ID = uuid.New()
	}

	// Set initial state
	batch.Status = domain.BatchStatusPending
	batch.CreatedAt = time.Now()
	batch.UpdatedAt = time.Now()
	batch.CompletedCount = 0
	batch.FailedCount = 0
	batch.CancelledCount = 0

	// Create in database
	if err := s.batchRepo.Create(ctx, batch); err != nil {
		return err
	}

	// Queue for processing (non-blocking)
	go s.processBatchQueue(ctx, batch.ID, batch.OrganizationID)

	return nil
}

// GetBatchSubmission retrieves a batch submission by ID
func (s *BatchSubmissionService) GetBatchSubmission(ctx context.Context, id uuid.UUID, orgID uuid.UUID) (*domain.BatchSubmission, error) {
	return s.batchRepo.GetByID(ctx, id, orgID)
}

// ListBatchSubmissions lists batch submissions for an organization
func (s *BatchSubmissionService) ListBatchSubmissions(ctx context.Context, orgID uuid.UUID, limit int, offset int) ([]*domain.BatchSubmission, int64, error) {
	return s.batchRepo.ListByOrganization(ctx, orgID, limit, offset)
}

// CancelBatchSubmission cancels an active batch submission
func (s *BatchSubmissionService) CancelBatchSubmission(ctx context.Context, id uuid.UUID, orgID uuid.UUID) error {
	// Get the batch
	batch, err := s.batchRepo.GetByID(ctx, id, orgID)
	if err != nil {
		return err
	}

	// Check if already terminal
	if batch.Status == domain.BatchStatusCompleted ||
		batch.Status == domain.BatchStatusFailed ||
		batch.Status == domain.BatchStatusCancelled {
		return errors.New("cannot cancel a batch in terminal state")
	}

	// Update status to cancelled
	if err := s.batchRepo.UpdateStatus(ctx, id, orgID, domain.BatchStatusCancelled); err != nil {
		return err
	}

	// Mark queue as inactive
	s.queueMutex.Lock()
	if q, exists := s.activeQueues[id]; exists {
		q.Active = false
	}
	s.queueMutex.Unlock()

	return nil
}

// RetryVideo retries a failed video within a batch
func (s *BatchSubmissionService) RetryVideo(ctx context.Context, videoID uuid.UUID, orgID uuid.UUID) error {
	// Get the video
	video, err := s.videoRepo.GetByID(ctx, videoID, orgID)
	if err != nil {
		return err
	}

	// Check if video is failed and created by batch
	if !video.CreatedByBatch {
		return errors.New("can only retry batch-created videos")
	}

	if video.GenerationStatus != domain.ShortVideoStatusFailed {
		return errors.New("can only retry failed videos")
	}

	// Get the batch to check retry limit
	if video.BatchID == nil {
		return errors.New("video not associated with batch")
	}

	batch, err := s.batchRepo.GetByID(ctx, *video.BatchID, orgID)
	if err != nil {
		return err
	}

	// Check retry limit
	if video.RetryCount >= batch.RetryLimit {
		return errors.New("retry limit exceeded")
	}

	// Increment retry count and reset status
	video.RetryCount++
	video.GenerationStatus = domain.ShortVideoStatusPending
	video.ErrorMessage = nil

	// Update in database
	if err := s.videoRepo.UpdateStatus(ctx, videoID, orgID, domain.ShortVideoStatusPending, nil, nil); err != nil {
		return err
	}

	// Re-submit for generation
	if s.productionSvc != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Convert parameters from json.RawMessage to map[string]interface{}
			var params map[string]interface{}
			if err := video.Parameters.UnmarshalJSON(video.Parameters); err != nil {
				return
			}

			_, _ = s.productionSvc.StartShortVideoGeneration(
				ctx,
				videoID.String(),
				video.HeyGenAvatarID,
				params,
				orgID.String(),
			)
		}()
	}

	return nil
}

// processBatchQueue processes videos in a batch submission
func (s *BatchSubmissionService) processBatchQueue(ctx context.Context, batchID uuid.UUID, orgID uuid.UUID) {
	// Get batch details
	batch, err := s.batchRepo.GetByID(ctx, batchID, orgID)
	if err != nil {
		return
	}

	// Update status to queued
	if err := s.batchRepo.UpdateStatus(ctx, batchID, orgID, domain.BatchStatusQueued); err != nil {
		return
	}

	// Register queue
	s.queueMutex.Lock()
	queue := &BatchQueue{
		BatchID:        batchID,
		OrganizationID: orgID,
		Concurrency:    batch.ConcurrencyLimit,
		RetryLimit:     batch.RetryLimit,
		Workers:        0,
		TotalCount:     batch.TotalCount,
		Active:         true,
	}
	s.activeQueues[batchID] = queue
	s.queueMutex.Unlock()

	// Mark as processing
	if err := s.batchRepo.UpdateStatus(ctx, batchID, orgID, domain.BatchStatusProcessing); err != nil {
		s.queueMutex.Lock()
		delete(s.activeQueues, batchID)
		s.queueMutex.Unlock()
		return
	}

	// TODO: Fetch batch items from database and process with concurrency control
	// This is a placeholder for the actual implementation

	// For now, just mark as completed
	if err := s.batchRepo.UpdateStatus(ctx, batchID, orgID, domain.BatchStatusCompleted); err != nil {
		return
	}

	// Cleanup
	s.queueMutex.Lock()
	delete(s.activeQueues, batchID)
	s.queueMutex.Unlock()
}

// UpdateBatchProgress updates progress counters for a batch
func (s *BatchSubmissionService) UpdateBatchProgress(ctx context.Context, batchID uuid.UUID, orgID uuid.UUID, completedCount, failedCount, cancelledCount int) error {
	return s.batchRepo.UpdateProgress(ctx, batchID, orgID, completedCount, failedCount, cancelledCount)
}
