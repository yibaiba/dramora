package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/service"
)

// Valid HeyGen avatar IDs
const (
	HeyGenAvatarProfessionalFemale = "avatar_001"
	HeyGenAvatarProfessionalMale   = "avatar_002"
	HeyGenAvatarYoungStyle         = "avatar_003"
)

// isValidHeyGenAvatarID validates if the avatar ID is supported.
func isValidHeyGenAvatarID(avatarID string) bool {
	switch avatarID {
	case HeyGenAvatarProfessionalFemale, HeyGenAvatarProfessionalMale, HeyGenAvatarYoungStyle:
		return true
	default:
		return false
	}
}

// GetDefaultHeyGenAvatarID returns the default avatar for short videos.
func GetDefaultHeyGenAvatarID() string {
	return HeyGenAvatarProfessionalFemale
}

// CreateShortVideoTemplateRequest represents the request body for creating a template
type CreateShortVideoTemplateRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Config      json.RawMessage `json:"config"`
}

// UpdateShortVideoTemplateRequest represents the request body for updating a template
type UpdateShortVideoTemplateRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Config      json.RawMessage `json:"config"`
}

// CreateShortVideoRequest represents the request body for creating a short video
type CreateShortVideoRequest struct {
	TemplateID     uuid.UUID       `json:"templateId"`
	Parameters     json.RawMessage `json:"parameters"`
	HeyGenAvatarID string          `json:"heyGenAvatarId"` // NEW: virtual presenter choice
}

// ShortVideoHandler handles HTTP requests for short videos
type ShortVideoHandler struct {
	templateRepo      repo.ShortVideoTemplateRepository
	videoRepo         repo.ShortVideoRepository
	productionService *service.ProductionService
}

// NewShortVideoHandler creates a new short video handler
func NewShortVideoHandler(
	templateRepo repo.ShortVideoTemplateRepository,
	videoRepo repo.ShortVideoRepository,
	productionService *service.ProductionService,
) *ShortVideoHandler {
	return &ShortVideoHandler{
		templateRepo:      templateRepo,
		videoRepo:         videoRepo,
		productionService: productionService,
	}
}

// RegisterRoutes registers all short video routes
func (h *ShortVideoHandler) RegisterRoutes(router chi.Router) {
	router.Route("/api/v1/short-video-templates", func(r chi.Router) {
		r.Post("/", h.CreateTemplate)
		r.Get("/", h.ListTemplates)
		r.Get("/{id}", h.GetTemplate)
		r.Delete("/{id}", h.DeleteTemplate)
	})

	router.Route("/api/v1/short-videos", func(r chi.Router) {
		r.Post("/", h.CreateShortVideo)
		r.Get("/", h.ListShortVideos)
		r.Get("/{id}", h.GetShortVideo)
		r.Delete("/{id}", h.DeleteShortVideo)
	})
}

// CreateTemplate handles POST /api/v1/short-video-templates
func (h *ShortVideoHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	orgID, ok := r.Context().Value("organizationID").(uuid.UUID)
	if !ok {
		http.Error(w, "missing organization context", http.StatusUnauthorized)
		return
	}

	var req CreateShortVideoTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Category == "" {
		http.Error(w, "category is required", http.StatusBadRequest)
		return
	}
	if len(req.Config) == 0 {
		http.Error(w, "config is required", http.StatusBadRequest)
		return
	}

	template := &domain.ShortVideoTemplate{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           req.Name,
		Description:    req.Description,
		Category:       req.Category,
		Config:         req.Config,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.templateRepo.Create(r.Context(), template); err != nil {
		http.Error(w, "failed to create template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(template)
}

// ListTemplates handles GET /api/v1/short-video-templates
func (h *ShortVideoHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	orgID, ok := r.Context().Value("organizationID").(uuid.UUID)
	if !ok {
		http.Error(w, "missing organization context", http.StatusUnauthorized)
		return
	}

	templates, err := h.templateRepo.ListByOrganization(r.Context(), orgID)
	if err != nil {
		http.Error(w, "failed to list templates", http.StatusInternalServerError)
		return
	}

	if templates == nil {
		templates = []*domain.ShortVideoTemplate{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// GetTemplate handles GET /api/v1/short-video-templates/{id}
func (h *ShortVideoHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	orgID, ok := r.Context().Value("organizationID").(uuid.UUID)
	if !ok {
		http.Error(w, "missing organization context", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid template id", http.StatusBadRequest)
		return
	}

	template, err := h.templateRepo.GetByID(r.Context(), id, orgID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "template not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to get template", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(template)
}

// DeleteTemplate handles DELETE /api/v1/short-video-templates/{id}
func (h *ShortVideoHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	orgID, ok := r.Context().Value("organizationID").(uuid.UUID)
	if !ok {
		http.Error(w, "missing organization context", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid template id", http.StatusBadRequest)
		return
	}

	if err := h.templateRepo.Delete(r.Context(), id, orgID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "template not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to delete template", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateShortVideo handles POST /api/v1/short-videos
func (h *ShortVideoHandler) CreateShortVideo(w http.ResponseWriter, r *http.Request) {
	orgID, ok := r.Context().Value("organizationID").(uuid.UUID)
	if !ok {
		http.Error(w, "missing organization context", http.StatusUnauthorized)
		return
	}

	var req CreateShortVideoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.TemplateID == uuid.Nil {
		http.Error(w, "templateId is required", http.StatusBadRequest)
		return
	}
	if len(req.Parameters) == 0 {
		http.Error(w, "parameters is required", http.StatusBadRequest)
		return
	}

	// Validate HeyGen avatar ID (use default if not provided)
	if req.HeyGenAvatarID == "" {
		req.HeyGenAvatarID = GetDefaultHeyGenAvatarID()
	} else if !isValidHeyGenAvatarID(req.HeyGenAvatarID) {
		http.Error(w, "invalid heyGenAvatarId: must be one of avatar_001, avatar_002, avatar_003", http.StatusBadRequest)
		return
	}

	// Verify template exists in the same organization
	_, err := h.templateRepo.GetByID(r.Context(), req.TemplateID, orgID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "template not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to verify template", http.StatusInternalServerError)
		}
		return
	}

	video := &domain.ShortVideo{
		ID:               uuid.New(),
		OrganizationID:   orgID,
		TemplateID:       req.TemplateID,
		Parameters:       req.Parameters,
		HeyGenAvatarID:   req.HeyGenAvatarID,
		Status:           domain.ShortVideoStatusPending,
		GenerationStatus: domain.ShortVideoStatusPending,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		Version:          1,
	}

	if err := h.videoRepo.Create(r.Context(), video); err != nil {
		http.Error(w, "failed to create short video", http.StatusInternalServerError)
		return
	}

	// Asynchronously start the generation process if production service is available
	if h.productionService != nil {
		go func() {
			// Decode parameters into a map for the service
			var params map[string]interface{}
			if err := json.Unmarshal(req.Parameters, &params); err != nil {
				// Log error but don't fail the request
				return
			}

			// Start video generation (will be updated to call HeyGen API in Phase 3c+)
			generatedVideoID, err := h.productionService.StartShortVideoGeneration(
				r.Context(),
				video.ID.String(),
				video.HeyGenAvatarID,
				params,
				video.OrganizationID.String(),
			)

			if err == nil {
				// Update video with generated ID and status
				video.HeyGenVideoID = generatedVideoID
				video.GenerationStatus = domain.ShortVideoStatusGenerating
				video.Status = domain.ShortVideoStatusGenerating
				video.UpdatedAt = time.Now()
				video.Version++
				// Note: In production, we would use an Update method here
				// For now, the status is stored in memory
			}
		}()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(video)
}

// ListShortVideos handles GET /api/v1/short-videos
func (h *ShortVideoHandler) ListShortVideos(w http.ResponseWriter, r *http.Request) {
	orgID, ok := r.Context().Value("organizationID").(uuid.UUID)
	if !ok {
		http.Error(w, "missing organization context", http.StatusUnauthorized)
		return
	}

	// Parse pagination parameters
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

	videos, total, err := h.videoRepo.ListByOrganization(r.Context(), orgID, limit, offset)
	if err != nil {
		http.Error(w, "failed to list short videos", http.StatusInternalServerError)
		return
	}

	if videos == nil {
		videos = []*domain.ShortVideo{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	json.NewEncoder(w).Encode(videos)
}

// GetShortVideo handles GET /api/v1/short-videos/{id}
func (h *ShortVideoHandler) GetShortVideo(w http.ResponseWriter, r *http.Request) {
	orgID, ok := r.Context().Value("organizationID").(uuid.UUID)
	if !ok {
		http.Error(w, "missing organization context", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid video id", http.StatusBadRequest)
		return
	}

	video, err := h.videoRepo.GetByID(r.Context(), id, orgID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "video not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to get video", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(video)
}

// DeleteShortVideo handles DELETE /api/v1/short-videos/{id}
func (h *ShortVideoHandler) DeleteShortVideo(w http.ResponseWriter, r *http.Request) {
	orgID, ok := r.Context().Value("organizationID").(uuid.UUID)
	if !ok {
		http.Error(w, "missing organization context", http.StatusUnauthorized)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid video id", http.StatusBadRequest)
		return
	}

	if err := h.videoRepo.Delete(r.Context(), id, orgID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			http.Error(w, "video not found", http.StatusNotFound)
		} else {
			http.Error(w, "failed to delete video", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
