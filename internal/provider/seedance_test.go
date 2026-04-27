package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildSeedanceGenerationRequestUsesImageReferences(t *testing.T) {
	request := BuildSeedanceGenerationRequest(SeedanceRequestInput{
		Prompt:      "SH001 opening shot",
		TaskType:    TaskTypeImageToVideo,
		DurationSec: 4,
		References: []SeedanceRefToken{
			{Token: "@image1", Role: "first_frame", URL: "manmu://ref/one"},
			{Token: "@image2", Role: "reference_image", URL: "manmu://ref/two"},
		},
	})

	if request.Model != ModelSeedance10ProFast {
		t.Fatalf("expected fast model, got %q", request.Model)
	}
	if request.Mode != string(TaskTypeImageToVideo) {
		t.Fatalf("expected image-to-video mode, got %q", request.Mode)
	}
	if len(request.Content) != 3 {
		t.Fatalf("expected text plus two image refs, got %+v", request.Content)
	}
	if request.ReferenceTokens[1].Token != "@image2" {
		t.Fatalf("expected second reference token, got %+v", request.ReferenceTokens)
	}
}

func TestSeedanceAdapterDefaultsToFakeModeWithoutKey(t *testing.T) {
	t.Setenv("ARK_API_KEY", "")
	t.Setenv("ARK_API_BASE_URL", "")

	adapter := NewSeedanceAdapterFromEnv()
	if adapter.Mode() != "fake" {
		t.Fatalf("expected fake mode without key, got %q", adapter.Mode())
	}
	if adapter.BaseURL() != DefaultSeedanceArkBaseURL {
		t.Fatalf("expected default base URL, got %q", adapter.BaseURL())
	}
}

func TestSeedanceAdapterSubmitsArkRequestWhenKeyPresent(t *testing.T) {
	var authHeader string
	var payload SeedanceGenerationRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("authorization")
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ark-task-1","status":"queued"}`))
	}))
	defer server.Close()

	adapter := NewSeedanceAdapter("test-key", server.URL, server.Client())
	task, err := adapter.SubmitGeneration(context.Background(), SeedanceRequestInput{
		Prompt:   "test",
		TaskType: TaskTypeTextToVideo,
	})
	if err != nil {
		t.Fatalf("submit generation: %v", err)
	}
	if task.Mode != "ark" || task.ID != "ark-task-1" {
		t.Fatalf("unexpected task: %+v", task)
	}
	if authHeader != "Bearer test-key" {
		t.Fatalf("expected bearer auth header, got %q", authHeader)
	}
	if payload.Model != ModelSeedance10ProFast {
		t.Fatalf("expected fast model payload, got %+v", payload)
	}
}
