package heygen

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/yibaiba/dramora/internal/provider"
)

// HeyGenAdapter implements the provider.Adapter interface for HeyGen video generation.
type HeyGenAdapter struct {
	client *Client
	logger *slog.Logger
}

// NewHeyGenAdapter creates a new HeyGen adapter.
func NewHeyGenAdapter(client *Client, logger *slog.Logger) *HeyGenAdapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &HeyGenAdapter{
		client: client,
		logger: logger,
	}
}

// Name returns the adapter name.
func (a *HeyGenAdapter) Name() string {
	return "heygen"
}

// Capabilities returns the capabilities of the HeyGen adapter.
func (a *HeyGenAdapter) Capabilities(ctx context.Context) ([]provider.Capability, error) {
	// HeyGen supports avatar-based video generation
	return []provider.Capability{
		{
			TaskType:           provider.TaskTypeAvatarVideo,
			MaxReferenceImages: 0,   // Not used for HeyGen
			MaxDurationSeconds: 300, // Up to 5 minutes
			SupportsCancel:     true,
		},
	}, nil
}

// GenerateVideo generates a short video using HeyGen.
type GenerateVideoRequest struct {
	Script          string   `json:"script"`
	AvatarID        AvatarID `json:"avatar_id"`
	VoiceID         string   `json:"voice_id,omitempty"`
	VoiceSpeed      float64  `json:"voice_speed,omitempty"`
	BackgroundMusic string   `json:"background_music,omitempty"`
}

// GenerateVideo submits a video generation request to HeyGen.
func (a *HeyGenAdapter) GenerateVideo(ctx context.Context, req *GenerateVideoRequest) (map[string]any, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	if req.Script == "" {
		return nil, fmt.Errorf("script cannot be empty")
	}

	if req.AvatarID == "" {
		return nil, fmt.Errorf("avatar_id cannot be empty")
	}

	// Set default voice speed
	voiceSpeed := req.VoiceSpeed
	if voiceSpeed == 0 {
		voiceSpeed = 1.0
	}

	videoReq := &VideoRequest{
		Script:             req.Script,
		AvatarID:           req.AvatarID,
		VoiceID:            req.VoiceID,
		VoiceSpeed:         voiceSpeed,
		BackgroundMusicURL: req.BackgroundMusic,
	}

	a.logger.Debug("submitting video generation request",
		"script_length", len(req.Script),
		"avatar_id", req.AvatarID,
	)

	resp, err := a.client.CreateVideo(ctx, videoReq)
	if err != nil {
		a.logger.Error("failed to create video", "error", err)
		return nil, fmt.Errorf("create video: %w", err)
	}

	a.logger.Info("video generation request submitted",
		"video_id", resp.VideoID,
		"status", resp.Status,
	)

	// Return result as map for flexibility
	result := map[string]any{
		"video_id":   resp.VideoID,
		"status":     resp.Status,
		"created_at": time.Now().Unix(),
	}

	if resp.VideoURL != "" {
		result["video_url"] = resp.VideoURL
	}

	return result, nil
}

// GetVideoStatus retrieves the current status of a video generation.
func (a *HeyGenAdapter) GetVideoStatus(ctx context.Context, videoID string) (map[string]any, error) {
	if videoID == "" {
		return nil, fmt.Errorf("videoID cannot be empty")
	}

	resp, err := a.client.GetVideoStatus(ctx, videoID)
	if err != nil {
		a.logger.Error("failed to get video status", "video_id", videoID, "error", err)
		return nil, fmt.Errorf("get video status: %w", err)
	}

	result := map[string]any{
		"video_id": resp.VideoID,
		"status":   resp.Status,
	}

	if resp.VideoURL != "" {
		result["video_url"] = resp.VideoURL
	}

	if resp.DurationSeconds > 0 {
		result["duration_seconds"] = resp.DurationSeconds
	}

	if resp.ErrorMessage != "" {
		result["error_message"] = resp.ErrorMessage
	}

	return result, nil
}

// CancelVideo cancels an in-progress video generation.
func (a *HeyGenAdapter) CancelVideo(ctx context.Context, videoID string) error {
	if videoID == "" {
		return fmt.Errorf("videoID cannot be empty")
	}

	err := a.client.CancelVideo(ctx, videoID)
	if err != nil {
		a.logger.Error("failed to cancel video", "video_id", videoID, "error", err)
		return fmt.Errorf("cancel video: %w", err)
	}

	a.logger.Info("video generation cancelled", "video_id", videoID)
	return nil
}

// MockHeyGenAdapter provides mock responses for testing without API calls.
type MockHeyGenAdapter struct {
	logger *slog.Logger
}

// NewMockHeyGenAdapter creates a new mock HeyGen adapter.
func NewMockHeyGenAdapter(logger *slog.Logger) *MockHeyGenAdapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &MockHeyGenAdapter{logger: logger}
}

// Name returns the adapter name.
func (a *MockHeyGenAdapter) Name() string {
	return "heygen-mock"
}

// Capabilities returns the capabilities of the mock adapter.
func (a *MockHeyGenAdapter) Capabilities(ctx context.Context) ([]provider.Capability, error) {
	return []provider.Capability{
		{
			TaskType:           provider.TaskTypeAvatarVideo,
			MaxReferenceImages: 0,
			MaxDurationSeconds: 300,
			SupportsCancel:     true,
		},
	}, nil
}

// GenerateVideo returns a mock response.
func (a *MockHeyGenAdapter) GenerateVideo(ctx context.Context, req *GenerateVideoRequest) (map[string]any, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	mockVideoID := fmt.Sprintf("mock-video-%d", time.Now().UnixMilli())

	a.logger.Debug("mock: video generation request",
		"video_id", mockVideoID,
		"script_length", len(req.Script),
	)

	return map[string]any{
		"video_id":   mockVideoID,
		"status":     "processing",
		"created_at": time.Now().Unix(),
	}, nil
}

// GetVideoStatus returns a mock status (simulating completed video).
func (a *MockHeyGenAdapter) GetVideoStatus(ctx context.Context, videoID string) (map[string]any, error) {
	if videoID == "" {
		return nil, fmt.Errorf("videoID cannot be empty")
	}

	return map[string]any{
		"video_id":         videoID,
		"status":           "completed",
		"video_url":        fmt.Sprintf("https://example.com/videos/%s.mp4", videoID),
		"duration_seconds": 30,
	}, nil
}

// CancelVideo returns nil for mock.
func (a *MockHeyGenAdapter) CancelVideo(ctx context.Context, videoID string) error {
	if videoID == "" {
		return fmt.Errorf("videoID cannot be empty")
	}

	a.logger.Debug("mock: video cancelled", "video_id", videoID)
	return nil
}
