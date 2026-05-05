package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/service"
)

// CreateBatchSubmissionRequest represents a request to create a batch submission
type CreateBatchSubmissionRequest struct {
	Name             string  `json:"name"`
	Description      *string `json:"description,omitempty"`
	TotalCount       int     `json:"totalCount"`
	ConcurrencyLimit int     `json:"concurrencyLimit"`
	RetryLimit       int     `json:"retryLimit"`
}

// BatchSubmissionResponse represents a batch submission in API responses
type BatchSubmissionResponse struct {
	ID               uuid.UUID `json:"id"`
	OrganizationID   uuid.UUID `json:"organizationId"`
	CreatedByUserID  uuid.UUID `json:"createdByUserId"`
	Name             string    `json:"name"`
	Description      *string   `json:"description,omitempty"`
	TotalCount       int       `json:"totalCount"`
	CompletedCount   int       `json:"completedCount"`
	FailedCount      int       `json:"failedCount"`
	CancelledCount   int       `json:"cancelledCount"`
	Status           string    `json:"status"`
	ConcurrencyLimit int       `json:"concurrencyLimit"`
	RetryLimit       int       `json:"retryLimit"`
	CreatedAt        string    `json:"createdAt"`
	UpdatedAt        string    `json:"updatedAt"`
	StartedAt        *string   `json:"startedAt,omitempty"`
	CompletedAt      *string   `json:"completedAt,omitempty"`
}

// BatchSubmissionHandler handles HTTP requests for batch submissions
type BatchSubmissionHandler struct {
	batchService *service.BatchSubmissionService
}

// NewBatchSubmissionHandler creates a new batch submission handler
func NewBatchSubmissionHandler(batchService *service.BatchSubmissionService) *BatchSubmissionHandler {
	return &BatchSubmissionHandler{
		batchService: batchService,
	}
}

// CreateBatchSubmission handles POST /api/v1/batch-submissions:create
func (h *BatchSubmissionHandler) CreateBatchSubmission(w http.ResponseWriter, r *http.Request) {
	// Extract auth context
	auth, ok := service.RequestAuthFromContext(r.Context())
	if !ok || auth.OrganizationID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "organization context required")
		return
	}

	orgID, err := uuid.Parse(auth.OrganizationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid organization ID")
		return
	}

	userID, err := uuid.Parse(auth.UserID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid user ID")
		return
	}

	// Parse request body
	var req CreateBatchSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	// Validate request
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "name is required")
		return
	}
	if req.TotalCount <= 0 || req.TotalCount > 1000 {
		writeError(w, http.StatusBadRequest, "invalid_request", "totalCount must be between 1 and 1000")
		return
	}
	if req.ConcurrencyLimit < 1 || req.ConcurrencyLimit > 8 {
		writeError(w, http.StatusBadRequest, "invalid_request", "concurrencyLimit must be between 1 and 8")
		return
	}
	if req.RetryLimit < 0 || req.RetryLimit > 5 {
		writeError(w, http.StatusBadRequest, "invalid_request", "retryLimit must be between 0 and 5")
		return
	}

	// Create batch submission
	batch := &domain.BatchSubmission{
		ID:               uuid.New(),
		OrganizationID:   orgID,
		CreatedByUserID:  userID,
		Name:             req.Name,
		Description:      req.Description,
		TotalCount:       req.TotalCount,
		ConcurrencyLimit: req.ConcurrencyLimit,
		RetryLimit:       req.RetryLimit,
		Status:           domain.BatchStatusPending,
	}

	if err := h.batchService.CreateBatchSubmission(r.Context(), batch); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to create batch submission")
		return
	}

	// Write response
	response := batchToResponse(batch)
	writeJSON(w, http.StatusCreated, response)
}

// GetBatchSubmission handles GET /api/v1/batch-submissions/{id}
func (h *BatchSubmissionHandler) GetBatchSubmission(w http.ResponseWriter, r *http.Request) {
	// Extract auth context
	auth, ok := service.RequestAuthFromContext(r.Context())
	if !ok || auth.OrganizationID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "organization context required")
		return
	}

	orgID, err := uuid.Parse(auth.OrganizationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid organization ID")
		return
	}

	// Extract batch ID from URL
	batchIDStr := chi.URLParam(r, "id")
	batchID, err := uuid.Parse(batchIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid batch ID")
		return
	}

	// Get batch submission
	batch, err := h.batchService.GetBatchSubmission(r.Context(), batchID, orgID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "batch submission not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get batch submission")
		return
	}

	// Write response
	response := batchToResponse(batch)
	writeJSON(w, http.StatusOK, response)
}

// ListBatchSubmissions handles GET /api/v1/batch-submissions
func (h *BatchSubmissionHandler) ListBatchSubmissions(w http.ResponseWriter, r *http.Request) {
	// Extract auth context
	auth, ok := service.RequestAuthFromContext(r.Context())
	if !ok || auth.OrganizationID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "organization context required")
		return
	}

	orgID, err := uuid.Parse(auth.OrganizationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid organization ID")
		return
	}

	// Parse query parameters
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// List batch submissions
	batches, total, err := h.batchService.ListBatchSubmissions(r.Context(), orgID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to list batch submissions")
		return
	}

	// Convert to response format
	responses := make([]BatchSubmissionResponse, len(batches))
	for i, batch := range batches {
		responses[i] = batchToResponse(batch)
	}

	// Write response with pagination info
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data":   responses,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// CancelBatchSubmission handles POST /api/v1/batch-submissions/{id}:cancel
func (h *BatchSubmissionHandler) CancelBatchSubmission(w http.ResponseWriter, r *http.Request) {
	// Extract auth context
	auth, ok := service.RequestAuthFromContext(r.Context())
	if !ok || auth.OrganizationID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "organization context required")
		return
	}

	orgID, err := uuid.Parse(auth.OrganizationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid organization ID")
		return
	}

	// Extract batch ID from URL
	batchIDStr := chi.URLParam(r, "id")
	batchID, err := uuid.Parse(batchIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid batch ID")
		return
	}

	// Cancel batch submission
	if err := h.batchService.CancelBatchSubmission(r.Context(), batchID, orgID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "batch submission not found")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Get updated batch to return
	batch, err := h.batchService.GetBatchSubmission(r.Context(), batchID, orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get batch submission")
		return
	}

	// Write response
	response := batchToResponse(batch)
	writeJSON(w, http.StatusOK, response)
}

// RetryVideo handles POST /api/v1/batch-submissions/{id}/videos/{videoId}:retry
func (h *BatchSubmissionHandler) RetryVideo(w http.ResponseWriter, r *http.Request) {
	// Extract auth context
	auth, ok := service.RequestAuthFromContext(r.Context())
	if !ok || auth.OrganizationID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "organization context required")
		return
	}

	orgID, err := uuid.Parse(auth.OrganizationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid organization ID")
		return
	}

	// Extract video ID from URL
	videoIDStr := chi.URLParam(r, "videoId")
	videoID, err := uuid.Parse(videoIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid video ID")
		return
	}

	// Retry video
	if err := h.batchService.RetryVideo(r.Context(), videoID, orgID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Write empty success response
	writeJSON(w, http.StatusNoContent, nil)
}

// Helper function to convert domain.BatchSubmission to API response
func batchToResponse(batch *domain.BatchSubmission) BatchSubmissionResponse {
	var startedAt, completedAt *string
	if batch.StartedAt != nil {
		s := batch.StartedAt.Format("2006-01-02T15:04:05Z07:00")
		startedAt = &s
	}
	if batch.CompletedAt != nil {
		c := batch.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
		completedAt = &c
	}

	return BatchSubmissionResponse{
		ID:               batch.ID,
		OrganizationID:   batch.OrganizationID,
		CreatedByUserID:  batch.CreatedByUserID,
		Name:             batch.Name,
		Description:      batch.Description,
		TotalCount:       batch.TotalCount,
		CompletedCount:   batch.CompletedCount,
		FailedCount:      batch.FailedCount,
		CancelledCount:   batch.CancelledCount,
		Status:           batch.Status,
		ConcurrencyLimit: batch.ConcurrencyLimit,
		RetryLimit:       batch.RetryLimit,
		CreatedAt:        batch.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        batch.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		StartedAt:        startedAt,
		CompletedAt:      completedAt,
	}
}
