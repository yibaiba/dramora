package provider

import (
	"context"
	"strings"
	"testing"
)

func TestNewImageProviderDispatch(t *testing.T) {
	for _, tc := range []struct {
		name       string
		cfg        CapabilityConfig
		wantPrefix string
		wantErr    bool
	}{
		{name: "default empty -> openai", cfg: CapabilityConfig{}, wantPrefix: "openai-image"},
		{name: "explicit openai", cfg: CapabilityConfig{ProviderType: "OpenAI"}, wantPrefix: "openai-image"},
		{name: "mock", cfg: CapabilityConfig{ProviderType: "mock"}, wantPrefix: "mock-image"},
		{name: "unknown", cfg: CapabilityConfig{ProviderType: "anthropic"}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewImageProvider(tc.cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got provider %v", p)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if !strings.HasPrefix(p.Name(), tc.wantPrefix) {
				t.Fatalf("name=%s want prefix %s", p.Name(), tc.wantPrefix)
			}
		})
	}
}

func TestNewVideoProviderDispatch(t *testing.T) {
	for _, tc := range []struct {
		name       string
		cfg        CapabilityConfig
		wantPrefix string
		wantErr    bool
	}{
		{name: "default -> seedance", cfg: CapabilityConfig{}, wantPrefix: "seedance-video"},
		{name: "mock", cfg: CapabilityConfig{ProviderType: "mock"}, wantPrefix: "mock-video"},
		{name: "unknown", cfg: CapabilityConfig{ProviderType: "openai"}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewVideoProvider(tc.cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if !strings.HasPrefix(p.Name(), tc.wantPrefix) {
				t.Fatalf("name=%s want prefix %s", p.Name(), tc.wantPrefix)
			}
		})
	}
}

func TestNewAudioProviderDispatch(t *testing.T) {
	for _, tc := range []struct {
		name       string
		cfg        CapabilityConfig
		wantPrefix string
		wantErr    bool
	}{
		{name: "default -> openai", cfg: CapabilityConfig{}, wantPrefix: "openai-audio"},
		{name: "mock", cfg: CapabilityConfig{ProviderType: "mock"}, wantPrefix: "mock-audio"},
		{name: "unknown", cfg: CapabilityConfig{ProviderType: "seedance"}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewAudioProvider(tc.cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if !strings.HasPrefix(p.Name(), tc.wantPrefix) {
				t.Fatalf("name=%s want prefix %s", p.Name(), tc.wantPrefix)
			}
		})
	}
}

func TestMockImageDeterministic(t *testing.T) {
	p, _ := NewImageProvider(CapabilityConfig{ProviderType: "mock", Model: "m1"})
	r1, err := p.Generate(context.Background(), ImageRequest{Prompt: "hello", NumImages: 2})
	if err != nil {
		t.Fatal(err)
	}
	r2, _ := p.Generate(context.Background(), ImageRequest{Prompt: "hello", NumImages: 2})
	if len(r1.URLs) != 2 || len(r2.URLs) != 2 {
		t.Fatalf("expected 2 URLs each, got %v / %v", r1.URLs, r2.URLs)
	}
	if r1.URLs[0] != r2.URLs[0] {
		t.Fatalf("expected deterministic, got %s vs %s", r1.URLs[0], r2.URLs[0])
	}
	if _, err := p.Generate(context.Background(), ImageRequest{Prompt: ""}); err == nil {
		t.Fatalf("expected empty-prompt error")
	}
}

func TestMockVideoLifecycle(t *testing.T) {
	p, _ := NewVideoProvider(CapabilityConfig{ProviderType: "mock"})
	task, err := p.Submit(context.Background(), VideoSubmitRequest{Prompt: "scene"})
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != "queued" {
		t.Fatalf("expected queued, got %s", task.Status)
	}
	polled, err := p.Poll(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if polled.Status != "succeeded" || polled.ResultURI == "" {
		t.Fatalf("unexpected poll: %+v", polled)
	}
	if _, err := p.Poll(context.Background(), ""); err == nil {
		t.Fatalf("expected empty-id error")
	}
}

func TestMockAudioDeterministic(t *testing.T) {
	p, _ := NewAudioProvider(CapabilityConfig{ProviderType: "mock"})
	r1, err := p.Synthesize(context.Background(), AudioRequest{Text: "你好", Voice: "alloy"})
	if err != nil {
		t.Fatal(err)
	}
	r2, _ := p.Synthesize(context.Background(), AudioRequest{Text: "你好", Voice: "alloy"})
	if r1.URL == "" || r1.URL != r2.URL {
		t.Fatalf("expected deterministic mock URL, got %s vs %s", r1.URL, r2.URL)
	}
	if _, err := p.Synthesize(context.Background(), AudioRequest{Text: ""}); err == nil {
		t.Fatalf("expected empty-text error")
	}
}
