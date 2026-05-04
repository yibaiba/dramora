package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ShortVideoTemplate represents a reusable template configuration for generating e-commerce short videos.
type ShortVideoTemplate struct {
	ID             uuid.UUID       `json:"id"`
	OrganizationID uuid.UUID       `json:"organizationId"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Category       string          `json:"category"` // e.g., "product-focus", "promotion", "usage-scenario"
	Config         json.RawMessage `json:"config"`   // Template parameter schema
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

// Validate checks if the template has valid required fields.
func (t *ShortVideoTemplate) Validate() error {
	if t.Name == "" {
		return ErrInvalidInput
	}
	if t.Category == "" {
		return ErrInvalidInput
	}
	if len(t.Config) == 0 {
		return ErrInvalidInput
	}
	return nil
}

// ShortVideoResult contains the output of a successful short video generation.
type ShortVideoResult struct {
	HeyGenVideoURL string `json:"heyGenVideoURL"`
	ThumbURL       string `json:"thumbURL"`
	Duration       int    `json:"duration"` // in seconds
	SizeBytes      int64  `json:"sizeBytes"`
}

// ShortVideo represents a generated or in-progress short video record.
type ShortVideo struct {
	ID             uuid.UUID         `json:"id"`
	OrganizationID uuid.UUID         `json:"organizationId"`
	TemplateID     uuid.UUID         `json:"templateId"`
	Parameters     json.RawMessage   `json:"parameters"`     // User input parameters
	HeyGenAvatarID string            `json:"heyGenAvatarId"` // HeyGen avatar choice
	HeyGenVideoID  string            `json:"heyGenVideoId,omitempty"` // HeyGen generated video ID
	GenerationStatus string          `json:"generationStatus"` // pending, generating, completed, failed
	Status         string            `json:"status"`         // pending, generating, completed, failed (deprecated, use generationStatus)
	ErrorMessage   *string           `json:"errorMessage,omitempty"`
	Result         *ShortVideoResult `json:"result,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
	Version        int               `json:"version"` // For optimistic locking
}

// Validate checks if the short video has valid required fields.
func (v *ShortVideo) Validate() error {
	if v.TemplateID == uuid.Nil {
		return ErrInvalidInput
	}
	if len(v.Parameters) == 0 {
		return ErrInvalidInput
	}
	if v.Status == "" {
		return ErrInvalidInput
	}
	return nil
}

// Status constants for short video generation
const (
	ShortVideoStatusPending    = "pending"
	ShortVideoStatusGenerating = "generating"
	ShortVideoStatusCompleted  = "completed"
	ShortVideoStatusFailed     = "failed"
)

// IsTerminalStatus returns true if the status is a terminal state.
func IsTerminalStatus(status string) bool {
	return status == ShortVideoStatusCompleted || status == ShortVideoStatusFailed
}

// CanTransitionTo returns true if a transition from current to next status is allowed.
func CanTransitionTo(current, next string) bool {
	transitions := map[string][]string{
		ShortVideoStatusPending:    {ShortVideoStatusGenerating},
		ShortVideoStatusGenerating: {ShortVideoStatusCompleted, ShortVideoStatusFailed},
		ShortVideoStatusCompleted:  {}, // Terminal state, no transitions
		ShortVideoStatusFailed:     {}, // Terminal state, no transitions
	}

	allowed, exists := transitions[current]
	if !exists {
		return false
	}

	for _, s := range allowed {
		if s == next {
			return true
		}
	}
	return false
}
