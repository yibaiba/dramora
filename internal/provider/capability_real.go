package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// openaiImageProvider 调用 OpenAI 兼容 /images/generations 端点，
// 与 LLM 适配器一致使用 BaseURL + APIKey + Model 三元组。
type openaiImageProvider struct {
	cfg    CapabilityConfig
	client *http.Client
}

func newOpenAIImage(cfg CapabilityConfig) *openaiImageProvider {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &openaiImageProvider{cfg: cfg, client: &http.Client{Timeout: timeout}}
}

func (p *openaiImageProvider) Name() string { return "openai-image" }

func (p *openaiImageProvider) Generate(ctx context.Context, req ImageRequest) (*ImageResult, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, fmt.Errorf("image prompt is required")
	}
	model := req.Model
	if model == "" {
		model = p.cfg.Model
	}
	body := map[string]any{
		"prompt": req.Prompt,
		"model":  model,
		"n":      maxInt(req.NumImages, 1),
	}
	if req.Width > 0 && req.Height > 0 {
		body["size"] = fmt.Sprintf("%dx%d", req.Width, req.Height)
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(p.cfg.BaseURL, "/") + "/images/generations"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai image request failed status=%d body=%s", resp.StatusCode, string(raw))
	}
	var parsed struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &parsed)
	urls := make([]string, 0, len(parsed.Data))
	for _, d := range parsed.Data {
		if d.URL != "" {
			urls = append(urls, d.URL)
		}
	}
	return &ImageResult{URLs: urls, Raw: string(raw)}, nil
}

// openaiAudioProvider 调用 OpenAI /audio/speech。返回字节流。
type openaiAudioProvider struct {
	cfg    CapabilityConfig
	client *http.Client
}

func newOpenAIAudio(cfg CapabilityConfig) *openaiAudioProvider {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &openaiAudioProvider{cfg: cfg, client: &http.Client{Timeout: timeout}}
}

func (p *openaiAudioProvider) Name() string { return "openai-audio" }

func (p *openaiAudioProvider) Synthesize(ctx context.Context, req AudioRequest) (*AudioResult, error) {
	if strings.TrimSpace(req.Text) == "" {
		return nil, fmt.Errorf("audio text is required")
	}
	model := req.Model
	if model == "" {
		model = p.cfg.Model
	}
	voice := req.Voice
	if voice == "" {
		voice = "alloy"
	}
	format := req.Format
	if format == "" {
		format = "mp3"
	}
	body := map[string]any{
		"model":           model,
		"input":           req.Text,
		"voice":           voice,
		"response_format": format,
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(p.cfg.BaseURL, "/") + "/audio/speech"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai audio request failed status=%d body=%s", resp.StatusCode, string(raw))
	}
	return &AudioResult{Bytes: raw}, nil
}

// seedanceVideoProvider 是现有 SeedanceAdapter 的 VideoProvider 包装，
// 让 video capability 也能通过统一工厂获取。
type seedanceVideoProvider struct {
	adapter *SeedanceAdapter
}

func newSeedanceVideo(cfg CapabilityConfig) *seedanceVideoProvider {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultSeedanceArkBaseURL
	}
	return &seedanceVideoProvider{adapter: NewSeedanceAdapter(cfg.APIKey, baseURL, client)}
}

func (p *seedanceVideoProvider) Name() string { return "seedance-video" }

func (p *seedanceVideoProvider) Submit(ctx context.Context, req VideoSubmitRequest) (VideoTask, error) {
	model := req.Model
	if model == "" {
		model = ModelSeedance10ProFast
	}
	input := SeedanceRequestInput{
		Prompt:      req.Prompt,
		TaskType:    TaskTypeTextToVideo,
		Ratio:       req.Ratio,
		Resolution:  req.Resolution,
		DurationSec: req.DurationSec,
		Seed:        req.Seed,
	}
	task, err := p.adapter.SubmitGeneration(ctx, input)
	if err != nil {
		return VideoTask{}, err
	}
	return VideoTask{ID: task.ID, Status: task.Status, Mode: task.Mode, ResultURI: task.ResultURI}, nil
}

func (p *seedanceVideoProvider) Poll(ctx context.Context, taskID string) (VideoTask, error) {
	task, err := p.adapter.PollGeneration(ctx, taskID)
	if err != nil {
		return VideoTask{}, err
	}
	return VideoTask{ID: task.ID, Status: task.Status, Mode: task.Mode, ResultURI: task.ResultURI}, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
