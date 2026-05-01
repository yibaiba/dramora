package provider

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
)

// mockImageProvider 不发网络请求，输出 deterministic 占位 URL，
// 方便测试与离线开发。
type mockImageProvider struct {
	model string
}

func newMockImage(cfg CapabilityConfig) *mockImageProvider {
	return &mockImageProvider{model: cfg.Model}
}

func (m *mockImageProvider) Name() string { return "mock-image" }

func (m *mockImageProvider) Generate(_ context.Context, req ImageRequest) (*ImageResult, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, fmt.Errorf("image prompt is required")
	}
	count := req.NumImages
	if count <= 0 {
		count = 1
	}
	urls := make([]string, 0, count)
	for i := 0; i < count; i++ {
		sum := sha1.Sum([]byte(fmt.Sprintf("%s|%d|%s", req.Prompt, i, m.model)))
		urls = append(urls, "manmu://providers/mock-image/"+hex.EncodeToString(sum[:8])+".png")
	}
	return &ImageResult{URLs: urls, Raw: "{\"mock\":true}"}, nil
}

// mockAudioProvider 同样不联网，返回 deterministic URL。
type mockAudioProvider struct {
	model string
	voice string
}

func newMockAudio(cfg CapabilityConfig) *mockAudioProvider {
	return &mockAudioProvider{model: cfg.Model}
}

func (m *mockAudioProvider) Name() string { return "mock-audio" }

func (m *mockAudioProvider) Synthesize(_ context.Context, req AudioRequest) (*AudioResult, error) {
	if strings.TrimSpace(req.Text) == "" {
		return nil, fmt.Errorf("audio text is required")
	}
	sum := sha1.Sum([]byte(req.Text + "|" + req.Voice + "|" + m.model))
	return &AudioResult{URL: "manmu://providers/mock-audio/" + hex.EncodeToString(sum[:8]) + ".mp3", Raw: "{\"mock\":true}"}, nil
}

// mockVideoProvider 用 Seedance 风格的 task 语义返回 deterministic 假 task。
type mockVideoProvider struct {
	model string
}

func newMockVideo(cfg CapabilityConfig) *mockVideoProvider {
	return &mockVideoProvider{model: cfg.Model}
}

func (m *mockVideoProvider) Name() string { return "mock-video" }

func (m *mockVideoProvider) Submit(_ context.Context, req VideoSubmitRequest) (VideoTask, error) {
	sum := sha1.Sum([]byte(req.Prompt + "|" + m.model))
	return VideoTask{ID: "mock-video-" + hex.EncodeToString(sum[:8]), Status: "queued", Mode: "fake"}, nil
}

func (m *mockVideoProvider) Poll(_ context.Context, taskID string) (VideoTask, error) {
	if strings.TrimSpace(taskID) == "" {
		return VideoTask{}, fmt.Errorf("task id is required")
	}
	return VideoTask{
		ID:        taskID,
		Status:    "succeeded",
		Mode:      "fake",
		ResultURI: "manmu://providers/mock-video/" + taskID + "/video.mp4",
	}, nil
}
