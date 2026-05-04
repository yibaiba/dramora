package jobs

import (
	"context"
	"encoding/json"
	"time"
)

// ShortVideoGenerationPayload represents a short video generation task.
type ShortVideoGenerationPayload struct {
	// Identifiers
	OrganizationID string `json:"organization_id"`
	ShortVideoID   string `json:"short_video_id"`
	TemplateID     string `json:"template_id"`

	// Generation parameters
	Parameters      json.RawMessage `json:"parameters"`                 // User input parameters
	Script          string          `json:"script"`                     // Generated script from Claude
	AvatarID        string          `json:"avatar_id"`                  // HeyGen avatar choice
	BackgroundMusic string          `json:"background_music,omitempty"` // Optional music URL

	// Internal tracking
	HeyGenVideoID string    `json:"heygen_video_id,omitempty"` // Set after HeyGen submission
	CreatedAt     time.Time `json:"created_at"`
	Attempts      int       `json:"attempts"`
	MaxRetries    int       `json:"max_retries"`
}

// HeyGenPollPayload represents a poll request for HeyGen video status.
type HeyGenPollPayload struct {
	OrganizationID  string    `json:"organization_id"`
	ShortVideoID    string    `json:"short_video_id"`
	HeyGenVideoID   string    `json:"heygen_video_id"`
	CreatedAt       time.Time `json:"created_at"`
	PollAttempts    int       `json:"poll_attempts"`
	MaxPollAttempts int       `json:"max_poll_attempts"`
}

// ShortVideoGenerationExecutor handles short video generation jobs.
type ShortVideoGenerationExecutor interface {
	// ProcessShortVideoGeneration processes a short video generation job.
	ProcessShortVideoGeneration(ctx context.Context, payload ShortVideoGenerationPayload) error

	// ProcessHeyGenPoll polls HeyGen for video generation status.
	ProcessHeyGenPoll(ctx context.Context, payload HeyGenPollPayload) error
}
