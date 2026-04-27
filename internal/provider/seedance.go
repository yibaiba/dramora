package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	ProviderSeedance          = "seedance"
	PresetSD2Fast             = "sd2_fast"
	ModelSeedance10ProFast    = "doubao-seedance-1-0-pro-fast-251015"
	DefaultSeedanceArkBaseURL = "https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks"
)

type ModelProfile struct {
	Provider           string
	Preset             string
	Model              string
	DefaultRatio       string
	DefaultResolution  string
	DefaultDurationSec int
	ServiceTier        string
	Capabilities       []Capability
}

type SeedanceAdapter struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

type SeedanceGenerationRequest struct {
	Model           string             `json:"model"`
	Content         []SeedanceContent  `json:"content"`
	Ratio           string             `json:"ratio"`
	Duration        int                `json:"duration"`
	Resolution      string             `json:"resolution"`
	CameraFixed     bool               `json:"camera_fixed"`
	Watermark       bool               `json:"watermark"`
	ReturnLastFrame bool               `json:"return_last_frame"`
	ServiceTier     string             `json:"service_tier"`
	Seed            int                `json:"seed,omitempty"`
	Mode            string             `json:"mode"`
	ReferenceTokens []SeedanceRefToken `json:"reference_tokens,omitempty"`
}

type SeedanceContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Role     string `json:"role,omitempty"`
}

type SeedanceRefToken struct {
	Token string `json:"token"`
	Role  string `json:"role"`
	URL   string `json:"url"`
}

type SeedanceRequestInput struct {
	Prompt      string
	TaskType    TaskType
	Ratio       string
	Resolution  string
	DurationSec int
	Seed        int
	References  []SeedanceRefToken
}

type SeedanceGenerationTask struct {
	ID     string
	Status string
	Mode   string
}

func NewSeedanceAdapterFromEnv() *SeedanceAdapter {
	return NewSeedanceAdapter(
		strings.TrimSpace(os.Getenv("ARK_API_KEY")),
		envOrDefault("ARK_API_BASE_URL", DefaultSeedanceArkBaseURL),
		http.DefaultClient,
	)
}

func NewSeedanceAdapter(apiKey string, baseURL string, client *http.Client) *SeedanceAdapter {
	if client == nil {
		client = http.DefaultClient
	}
	return &SeedanceAdapter{
		apiKey:  strings.TrimSpace(apiKey),
		baseURL: defaultString(baseURL, DefaultSeedanceArkBaseURL),
		client:  client,
	}
}

func SeedanceFastProfile() ModelProfile {
	return ModelProfile{
		Provider:           ProviderSeedance,
		Preset:             PresetSD2Fast,
		Model:              ModelSeedance10ProFast,
		DefaultRatio:       "16:9",
		DefaultResolution:  "720p",
		DefaultDurationSec: 5,
		ServiceTier:        "fast",
		Capabilities: []Capability{
			{TaskType: TaskTypeTextToVideo, MaxReferenceImages: 0, MaxDurationSeconds: 15, SupportsCancel: true},
			{TaskType: TaskTypeImageToVideo, MaxReferenceImages: 9, MaxDurationSeconds: 15, SupportsCancel: true},
			{TaskType: TaskTypeFirstLast, MaxReferenceImages: 2, MaxDurationSeconds: 15, SupportsCancel: true},
		},
	}
}

func (a *SeedanceAdapter) Name() string {
	return ProviderSeedance
}

func (a *SeedanceAdapter) Capabilities(context.Context) ([]Capability, error) {
	return SeedanceFastProfile().Capabilities, nil
}

func (a *SeedanceAdapter) Mode() string {
	if a.apiKey == "" {
		return "fake"
	}
	return "ark"
}

func (a *SeedanceAdapter) BaseURL() string {
	return a.baseURL
}

func (a *SeedanceAdapter) SubmitGeneration(
	ctx context.Context,
	input SeedanceRequestInput,
) (SeedanceGenerationTask, error) {
	if a.Mode() == "fake" {
		return SeedanceGenerationTask{ID: "fake-seedance-task", Status: "queued", Mode: "fake"}, nil
	}
	request := BuildSeedanceGenerationRequest(input)
	taskID, status, err := a.submitArkGeneration(ctx, request)
	if err != nil {
		return SeedanceGenerationTask{}, err
	}
	return SeedanceGenerationTask{ID: taskID, Status: status, Mode: "ark"}, nil
}

func (a *SeedanceAdapter) PollGeneration(ctx context.Context, taskID string) (SeedanceGenerationTask, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return SeedanceGenerationTask{}, fmt.Errorf("seedance task id is required")
	}
	if a.Mode() == "fake" {
		return SeedanceGenerationTask{ID: taskID, Status: "succeeded", Mode: "fake"}, nil
	}
	status, err := a.pollArkGeneration(ctx, taskID)
	if err != nil {
		return SeedanceGenerationTask{}, err
	}
	return SeedanceGenerationTask{ID: taskID, Status: status, Mode: "ark"}, nil
}

func BuildSeedanceGenerationRequest(input SeedanceRequestInput) SeedanceGenerationRequest {
	profile := SeedanceFastProfile()
	duration := input.DurationSec
	if duration <= 0 {
		duration = profile.DefaultDurationSec
	}
	return SeedanceGenerationRequest{
		Model:           profile.Model,
		Content:         seedanceContent(input),
		Ratio:           defaultString(input.Ratio, profile.DefaultRatio),
		Duration:        duration,
		Resolution:      defaultString(input.Resolution, profile.DefaultResolution),
		CameraFixed:     false,
		Watermark:       false,
		ReturnLastFrame: true,
		ServiceTier:     profile.ServiceTier,
		Seed:            input.Seed,
		Mode:            string(input.TaskType),
		ReferenceTokens: input.References,
	}
}

func seedanceContent(input SeedanceRequestInput) []SeedanceContent {
	content := []SeedanceContent{{Type: "text", Text: input.Prompt}}
	for _, ref := range input.References {
		content = append(content, SeedanceContent{
			Type:     "image_url",
			ImageURL: ref.URL,
			Role:     ref.Role,
		})
	}
	return content
}

func (a *SeedanceAdapter) submitArkGeneration(
	ctx context.Context,
	payload SeedanceGenerationRequest,
) (string, string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL, bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	request.Header.Set("authorization", "Bearer "+a.apiKey)
	request.Header.Set("content-type", "application/json")
	response, err := a.client.Do(request)
	if err != nil {
		return "", "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", "", fmt.Errorf("seedance ark request failed with status %d", response.StatusCode)
	}
	return decodeArkTask(response)
}

func (a *SeedanceAdapter) pollArkGeneration(ctx context.Context, taskID string) (string, error) {
	taskURL := strings.TrimRight(a.baseURL, "/") + "/" + url.PathEscape(taskID)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, taskURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("authorization", "Bearer "+a.apiKey)
	response, err := a.client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("seedance ark poll failed with status %d", response.StatusCode)
	}
	_, status, err := decodeArkTask(response)
	return status, err
}

func decodeArkTask(response *http.Response) (string, string, error) {
	var payload struct {
		ID     string `json:"id"`
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", "", err
	}
	id := defaultString(payload.ID, payload.TaskID)
	if id == "" {
		return "", "", fmt.Errorf("seedance ark response missing task id")
	}
	return id, defaultString(payload.Status, "submitted"), nil
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
