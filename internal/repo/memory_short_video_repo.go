package repo

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yibaiba/dramora/internal/domain"
)

// MemoryShortVideoTemplateRepository implements ShortVideoTemplateRepository using in-memory storage
type MemoryShortVideoTemplateRepository struct {
	mu        sync.RWMutex
	templates map[uuid.UUID]*domain.ShortVideoTemplate
}

// NewMemoryShortVideoTemplateRepository creates a new in-memory template repository
func NewMemoryShortVideoTemplateRepository() *MemoryShortVideoTemplateRepository {
	return &MemoryShortVideoTemplateRepository{
		templates: make(map[uuid.UUID]*domain.ShortVideoTemplate),
	}
}

// Create inserts a new template
func (r *MemoryShortVideoTemplateRepository) Create(ctx context.Context, template *domain.ShortVideoTemplate) error {
	if err := template.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.templates[template.ID] = template
	return nil
}

// GetByID retrieves a template by ID with organization permission check
func (r *MemoryShortVideoTemplateRepository) GetByID(ctx context.Context, id uuid.UUID, orgID uuid.UUID) (*domain.ShortVideoTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	template, exists := r.templates[id]
	if !exists || template.OrganizationID != orgID {
		return nil, domain.ErrNotFound
	}

	return template, nil
}

// ListByOrganization retrieves all templates for an organization
func (r *MemoryShortVideoTemplateRepository) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]*domain.ShortVideoTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var templates []*domain.ShortVideoTemplate
	for _, t := range r.templates {
		if t.OrganizationID == orgID {
			templates = append(templates, t)
		}
	}

	return templates, nil
}

// Update updates an existing template
func (r *MemoryShortVideoTemplateRepository) Update(ctx context.Context, template *domain.ShortVideoTemplate) error {
	if err := template.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.templates[template.ID]
	if !exists || existing.OrganizationID != template.OrganizationID {
		return domain.ErrNotFound
	}

	r.templates[template.ID] = template
	return nil
}

// Delete removes a template
func (r *MemoryShortVideoTemplateRepository) Delete(ctx context.Context, id uuid.UUID, orgID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.templates[id]
	if !exists || existing.OrganizationID != orgID {
		return domain.ErrNotFound
	}

	delete(r.templates, id)
	return nil
}

// MemoryShortVideoRepository implements ShortVideoRepository using in-memory storage
type MemoryShortVideoRepository struct {
	mu     sync.RWMutex
	videos map[uuid.UUID]*domain.ShortVideo
}

// NewMemoryShortVideoRepository creates a new in-memory short video repository
func NewMemoryShortVideoRepository() *MemoryShortVideoRepository {
	return &MemoryShortVideoRepository{
		videos: make(map[uuid.UUID]*domain.ShortVideo),
	}
}

// Create inserts a new short video
func (r *MemoryShortVideoRepository) Create(ctx context.Context, video *domain.ShortVideo) error {
	if err := video.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.videos[video.ID] = video
	return nil
}

// GetByID retrieves a short video by ID with organization permission check
func (r *MemoryShortVideoRepository) GetByID(ctx context.Context, id uuid.UUID, orgID uuid.UUID) (*domain.ShortVideo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	video, exists := r.videos[id]
	if !exists || video.OrganizationID != orgID {
		return nil, domain.ErrNotFound
	}

	return video, nil
}

// ListByOrganization retrieves all short videos for an organization with pagination
func (r *MemoryShortVideoRepository) ListByOrganization(ctx context.Context, orgID uuid.UUID, limit int, offset int) ([]*domain.ShortVideo, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var videos []*domain.ShortVideo
	for _, v := range r.videos {
		if v.OrganizationID == orgID {
			videos = append(videos, v)
		}
	}

	total := int64(len(videos))

	// Apply pagination
	if offset >= len(videos) {
		return []*domain.ShortVideo{}, total, nil
	}

	end := offset + limit
	if end > len(videos) {
		end = len(videos)
	}

	return videos[offset:end], total, nil
}

// UpdateStatus updates the status of a short video
func (r *MemoryShortVideoRepository) UpdateStatus(ctx context.Context, id uuid.UUID, orgID uuid.UUID, status string, errorMsg *string, result *domain.ShortVideoResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	video, exists := r.videos[id]
	if !exists || video.OrganizationID != orgID {
		return domain.ErrNotFound
	}

	if !domain.CanTransitionTo(video.Status, status) {
		return domain.ErrInvalidInput
	}

	video.Status = status
	video.ErrorMessage = errorMsg
	video.Result = result
	video.UpdatedAt = time.Now()
	video.Version++

	return nil
}

// Delete removes a short video
func (r *MemoryShortVideoRepository) Delete(ctx context.Context, id uuid.UUID, orgID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	video, exists := r.videos[id]
	if !exists || video.OrganizationID != orgID {
		return domain.ErrNotFound
	}

	delete(r.videos, id)
	return nil
}
