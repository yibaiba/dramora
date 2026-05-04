package heygen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		opts    *ClientOptions
		wantErr bool
	}{
		{
			name:    "valid client with defaults",
			apiKey:  "test-key",
			opts:    nil,
			wantErr: false,
		},
		{
			name:   "valid client with custom options",
			apiKey: "test-key",
			opts: &ClientOptions{
				BaseURL: "https://custom.example.com",
				Timeout: 60 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.apiKey, tt.opts)
			if client == nil {
				t.Fatal("NewClient returned nil")
			}
		})
	}
}

func TestCreateVideo_ValidationErrors(t *testing.T) {
	client := NewClient("test-key", nil)

	tests := []struct {
		name    string
		req     *VideoRequest
		wantErr string
	}{
		{
			name:    "nil request",
			req:     nil,
			wantErr: "request cannot be nil",
		},
		{
			name: "empty script",
			req: &VideoRequest{
				AvatarID: AvatarProfessionalFemale,
			},
			wantErr: "script cannot be empty",
		},
		{
			name: "empty avatar_id",
			req: &VideoRequest{
				Script: "Hello world",
			},
			wantErr: "avatar_id cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := client.CreateVideo(ctx, tt.req)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("want %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestCreateVideo_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("want POST, got %s", r.Method)
		}

		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "test-key" {
			t.Errorf("want api key 'test-key', got %q", apiKey)
		}

		// Mock successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{
			"video_id": "vid-123",
			"status": "processing",
			"created_at": "2025-01-01T00:00:00Z"
		}`))
	}))
	defer server.Close()

	client := NewClient("test-key", &ClientOptions{
		BaseURL:    server.URL,
		HTTPClient: &http.Client{},
		Timeout:    5 * time.Second,
	})

	req := &VideoRequest{
		Script:     "Welcome to our product showcase",
		AvatarID:   AvatarProfessionalFemale,
		VoiceSpeed: 1.0,
	}

	resp, err := client.CreateVideo(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateVideo failed: %v", err)
	}

	if resp.VideoID != "vid-123" {
		t.Errorf("want video_id 'vid-123', got %q", resp.VideoID)
	}

	if resp.Status != "processing" {
		t.Errorf("want status 'processing', got %q", resp.Status)
	}
}

func TestGetVideoStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("want GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"video_id": "vid-123",
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

	resp, err := client.GetVideoStatus(context.Background(), "vid-123")
	if err != nil {
		t.Fatalf("GetVideoStatus failed: %v", err)
	}

	if resp.Status != "completed" {
		t.Errorf("want status 'completed', got %q", resp.Status)
	}

	if resp.DurationSeconds != 30 {
		t.Errorf("want duration 30, got %d", resp.DurationSeconds)
	}
}

func TestGetVideoStatus_ValidationErrors(t *testing.T) {
	client := NewClient("test-key", nil)

	_, err := client.GetVideoStatus(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty videoID, got nil")
	}
	if err.Error() != "videoID cannot be empty" {
		t.Errorf("want 'videoID cannot be empty', got %q", err.Error())
	}
}

func TestCancelVideo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("want POST, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-key", &ClientOptions{
		BaseURL:    server.URL,
		HTTPClient: &http.Client{},
		Timeout:    5 * time.Second,
	})

	err := client.CancelVideo(context.Background(), "vid-123")
	if err != nil {
		t.Fatalf("CancelVideo failed: %v", err)
	}
}

func TestCancelVideo_ValidationErrors(t *testing.T) {
	client := NewClient("test-key", nil)

	err := client.CancelVideo(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty videoID, got nil")
	}
	if err.Error() != "videoID cannot be empty" {
		t.Errorf("want 'videoID cannot be empty', got %q", err.Error())
	}
}
