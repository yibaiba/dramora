package heygen

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AvatarID represents a HeyGen avatar (virtual presenter).
type AvatarID string

const (
	// HeyGen avatar IDs for different presenter styles
	AvatarProfessionalFemale AvatarID = "avatar_001" // Professional female presenter
	AvatarProfessionalMale   AvatarID = "avatar_002" // Professional male presenter
	AvatarYoungStyle         AvatarID = "avatar_003" // Young, trendy style
)

// VideoRequest represents a request to generate a video from text.
type VideoRequest struct {
	// The text/script to be spoken by the avatar
	Script string `json:"script"`
	// The avatar to use for the video
	AvatarID AvatarID `json:"avatar_id"`
	// Optional: voice settings (voice ID, speed, etc.)
	VoiceID string `json:"voice_id,omitempty"`
	// Optional: voice speed (0.5 to 2.0)
	VoiceSpeed float64 `json:"voice_speed,omitempty"`
	// Optional: background music URL
	BackgroundMusicURL string `json:"background_music_url,omitempty"`
}

// VideoResponse represents a response from HeyGen API after submitting a video request.
type VideoResponse struct {
	// Unique video ID
	VideoID string `json:"video_id"`
	// Current status of the video generation
	Status string `json:"status"` // "pending", "processing", "completed", "failed"
	// URL to access the video (populated when status is "completed")
	VideoURL string `json:"video_url,omitempty"`
	// Video duration in seconds (populated when completed)
	DurationSeconds int `json:"duration_seconds,omitempty"`
	// Error message if status is "failed"
	ErrorMessage string `json:"error_message,omitempty"`
	// Timestamp when the request was created
	CreatedAt string `json:"created_at,omitempty"`
	// Timestamp when the video was last updated
	UpdatedAt string `json:"updated_at,omitempty"`
}

// ClientOptions configures the HeyGen client.
type ClientOptions struct {
	BaseURL    string
	HTTPClient *http.Client
	Timeout    time.Duration
}

// Client handles communication with HeyGen API.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewClient creates a new HeyGen API client.
func NewClient(apiKey string, opts *ClientOptions) *Client {
	if opts == nil {
		opts = &ClientOptions{}
	}

	if opts.BaseURL == "" {
		opts.BaseURL = "https://api.heygen.com/v1"
	}

	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{}
	}

	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}

	return &Client{
		apiKey:     apiKey,
		baseURL:    opts.BaseURL,
		httpClient: opts.HTTPClient,
		timeout:    opts.Timeout,
	}
}

// CreateVideo submits a video generation request to HeyGen API.
func (c *Client) CreateVideo(ctx context.Context, req *VideoRequest) (*VideoResponse, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	if req.Script == "" {
		return nil, errors.New("script cannot be empty")
	}

	if req.AvatarID == "" {
		return nil, errors.New("avatar_id cannot be empty")
	}

	// Set default voice speed if not specified
	if req.VoiceSpeed == 0 {
		req.VoiceSpeed = 1.0
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/video/generate"

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("X-API-Key", c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	// Set timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.httpClient.Do(httpReq.WithContext(timeoutCtx))
	if err != nil {
		return nil, fmt.Errorf("api call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(body))
	}

	var videoResp VideoResponse
	if err := json.NewDecoder(resp.Body).Decode(&videoResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &videoResp, nil
}

// GetVideoStatus retrieves the current status of a video generation request.
func (c *Client) GetVideoStatus(ctx context.Context, videoID string) (*VideoResponse, error) {
	if videoID == "" {
		return nil, errors.New("videoID cannot be empty")
	}

	url := fmt.Sprintf("%s/video/%s", c.baseURL, videoID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("X-API-Key", c.apiKey)

	// Set timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.httpClient.Do(httpReq.WithContext(timeoutCtx))
	if err != nil {
		return nil, fmt.Errorf("api call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(body))
	}

	var videoResp VideoResponse
	if err := json.NewDecoder(resp.Body).Decode(&videoResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &videoResp, nil
}

// CancelVideo cancels an in-progress video generation request.
func (c *Client) CancelVideo(ctx context.Context, videoID string) error {
	if videoID == "" {
		return errors.New("videoID cannot be empty")
	}

	url := fmt.Sprintf("%s/video/%s/cancel", c.baseURL, videoID)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("X-API-Key", c.apiKey)

	// Set timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.httpClient.Do(httpReq.WithContext(timeoutCtx))
	if err != nil {
		return fmt.Errorf("api call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
