package heygen

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yibaiba/dramora/internal/provider"
)

func TestHeyGenAdapterName(t *testing.T) {
	adapter := NewHeyGenAdapter(NewClient("test", nil), slog.Default())
	if adapter.Name() != "heygen" {
		t.Errorf("want 'heygen', got %q", adapter.Name())
	}
}

func TestHeyGenAdapterCapabilities(t *testing.T) {
	adapter := NewHeyGenAdapter(NewClient("test", nil), slog.Default())

	caps, err := adapter.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities failed: %v", err)
	}

	if len(caps) != 1 {
		t.Errorf("want 1 capability, got %d", len(caps))
	}

	if caps[0].TaskType != provider.TaskTypeAvatarVideo {
		t.Errorf("want TaskTypeAvatarVideo, got %v", caps[0].TaskType)
	}

	if caps[0].MaxDurationSeconds != 300 {
		t.Errorf("want max duration 300, got %d", caps[0].MaxDurationSeconds)
	}

	if !caps[0].SupportsCancel {
		t.Error("want SupportsCancel true, got false")
	}
}

func TestHeyGenAdapterGenerateVideo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{
			"video_id": "vid-test",
			"status": "processing"
		}`))
	}))
	defer server.Close()

	client := NewClient("test-key", &ClientOptions{
		BaseURL:    server.URL,
		HTTPClient: &http.Client{},
		Timeout:    5 * time.Second,
	})

	adapter := NewHeyGenAdapter(client, slog.Default())

	req := &GenerateVideoRequest{
		Script:   "This is a test script",
		AvatarID: AvatarProfessionalFemale,
	}

	result, err := adapter.GenerateVideo(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateVideo failed: %v", err)
	}

	if result["video_id"] != "vid-test" {
		t.Errorf("want video_id 'vid-test', got %v", result["video_id"])
	}

	if result["status"] != "processing" {
		t.Errorf("want status 'processing', got %v", result["status"])
	}
}

func TestHeyGenAdapterGetVideoStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"video_id": "vid-test",
			"status": "completed",
			"video_url": "https://example.com/video.mp4",
			"duration_seconds": 30
		}`))
	}))
	defer server.Close()

	client := NewClient("test-key", &ClientOptions{
		BaseURL:    server.URL,
		HTTPClient: &http.Client{},
		Timeout:    5 * time.Second,
	})

	adapter := NewHeyGenAdapter(client, slog.Default())

	result, err := adapter.GetVideoStatus(context.Background(), "vid-test")
	if err != nil {
		t.Fatalf("GetVideoStatus failed: %v", err)
	}

	if result["status"] != "completed" {
		t.Errorf("want status 'completed', got %v", result["status"])
	}

	if result["duration_seconds"] != 30 {
		t.Errorf("want duration 30, got %v", result["duration_seconds"])
	}
}

func TestMockHeyGenAdapterName(t *testing.T) {
	adapter := NewMockHeyGenAdapter(slog.Default())
	if adapter.Name() != "heygen-mock" {
		t.Errorf("want 'heygen-mock', got %q", adapter.Name())
	}
}

func TestMockHeyGenAdapterGenerateVideo(t *testing.T) {
	adapter := NewMockHeyGenAdapter(slog.Default())

	req := &GenerateVideoRequest{
		Script:   "This is a test script",
		AvatarID: AvatarProfessionalMale,
	}

	result, err := adapter.GenerateVideo(context.Background(), req)
	if err != nil {
		t.Fatalf("GenerateVideo failed: %v", err)
	}

	if _, ok := result["video_id"]; !ok {
		t.Error("missing video_id in result")
	}

	if result["status"] != "processing" {
		t.Errorf("want status 'processing', got %v", result["status"])
	}
}

func TestMockHeyGenAdapterGetVideoStatus(t *testing.T) {
	adapter := NewMockHeyGenAdapter(slog.Default())

	result, err := adapter.GetVideoStatus(context.Background(), "vid-test")
	if err != nil {
		t.Fatalf("GetVideoStatus failed: %v", err)
	}

	if result["status"] != "completed" {
		t.Errorf("want status 'completed', got %v", result["status"])
	}

	if _, ok := result["video_url"]; !ok {
		t.Error("missing video_url in result")
	}
}
