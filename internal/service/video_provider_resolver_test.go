package service

import (
	"context"
	"testing"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/provider"
	"github.com/yibaiba/dramora/internal/repo"
)

// stubVideoProviderRepo lets tests inject a specific ProviderConfig for video capability.
type stubVideoProviderRepo struct {
	cfg domain.ProviderConfig
	err error
}

func (s *stubVideoProviderRepo) ListProviderConfigs(_ context.Context) ([]domain.ProviderConfig, error) {
	return nil, nil
}

func (s *stubVideoProviderRepo) GetProviderConfig(_ context.Context, capability string) (domain.ProviderConfig, error) {
	if s.err != nil {
		return domain.ProviderConfig{}, s.err
	}
	if capability != "video" {
		return domain.ProviderConfig{}, domain.ErrNotFound
	}
	return s.cfg, nil
}

func (s *stubVideoProviderRepo) SaveProviderConfig(_ context.Context, _ repo.SaveProviderConfigParams) (domain.ProviderConfig, error) {
	return domain.ProviderConfig{}, nil
}

// markerSeedance is a tagged seedanceProvider used to verify fallback path.
type markerSeedance struct{ tag string }

func (markerSeedance) SubmitGeneration(_ context.Context, _ provider.SeedanceRequestInput) (provider.SeedanceGenerationTask, error) {
	return provider.SeedanceGenerationTask{ID: "fallback"}, nil
}
func (markerSeedance) PollGeneration(_ context.Context, _ string) (provider.SeedanceGenerationTask, error) {
	return provider.SeedanceGenerationTask{}, nil
}

func TestResolveSeedanceFallsBackWithoutProviderService(t *testing.T) {
	prod := &ProductionService{seedance: markerSeedance{tag: "fallback"}}
	got := prod.resolveSeedance(context.Background())
	if _, ok := got.(markerSeedance); !ok {
		t.Fatalf("expected fallback markerSeedance, got %T", got)
	}
}

func TestResolveSeedanceUsesMockWhenConfigured(t *testing.T) {
	repo := &stubVideoProviderRepo{cfg: domain.ProviderConfig{Capability: "video", ProviderType: "mock"}}
	prod := &ProductionService{
		seedance:    markerSeedance{tag: "fallback"},
		providerSvc: NewProviderService(repo),
	}
	got := prod.resolveSeedance(context.Background())
	if _, ok := got.(mockSeedanceProvider); !ok {
		t.Fatalf("expected mockSeedanceProvider, got %T", got)
	}
	// confirm mock works end-to-end
	task, err := got.SubmitGeneration(context.Background(), provider.SeedanceRequestInput{Prompt: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != "queued" || task.ID == "" {
		t.Fatalf("unexpected mock task: %+v", task)
	}
	polled, err := got.PollGeneration(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if polled.Status != "succeeded" || polled.ResultURI == "" {
		t.Fatalf("unexpected poll: %+v", polled)
	}
}

func TestResolveSeedanceConstructsAdapterFromConfig(t *testing.T) {
	repo := &stubVideoProviderRepo{cfg: domain.ProviderConfig{
		Capability:   "video",
		ProviderType: "seedance",
		BaseURL:      "https://example.test/api",
		APIKey:       "test-key",
	}}
	prod := &ProductionService{
		seedance:    markerSeedance{tag: "fallback"},
		providerSvc: NewProviderService(repo),
	}
	got := prod.resolveSeedance(context.Background())
	if _, ok := got.(*provider.SeedanceAdapter); !ok {
		t.Fatalf("expected *provider.SeedanceAdapter, got %T", got)
	}
}

func TestResolveSeedanceFallsBackOnMissingConfig(t *testing.T) {
	repo := &stubVideoProviderRepo{err: domain.ErrNotFound}
	prod := &ProductionService{
		seedance:    markerSeedance{tag: "fallback"},
		providerSvc: NewProviderService(repo),
	}
	got := prod.resolveSeedance(context.Background())
	if _, ok := got.(markerSeedance); !ok {
		t.Fatalf("expected fallback when ErrNotFound, got %T", got)
	}
}
