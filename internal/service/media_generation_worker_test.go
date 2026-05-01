package service

import (
	"context"
	"strings"
	"testing"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/jobs"
	"github.com/yibaiba/dramora/internal/media"
	"github.com/yibaiba/dramora/internal/provider"
	"github.com/yibaiba/dramora/internal/repo"
)

// capabilityStubProviderRepo is a minimal ProviderConfigRepository that returns
// the configured (capability, provider_type) entry; everything else returns NotFound.
type capabilityStubProviderRepo struct {
	capability   string
	providerType string
}

func (r *capabilityStubProviderRepo) ListProviderConfigs(_ context.Context) ([]domain.ProviderConfig, error) {
	return nil, nil
}

func (r *capabilityStubProviderRepo) GetProviderConfig(_ context.Context, capability string) (domain.ProviderConfig, error) {
	if capability != r.capability {
		return domain.ProviderConfig{}, domain.ErrNotFound
	}
	return domain.ProviderConfig{Capability: r.capability, ProviderType: r.providerType}, nil
}

func (r *capabilityStubProviderRepo) SaveProviderConfig(_ context.Context, _ repo.SaveProviderConfigParams) (domain.ProviderConfig, error) {
	return domain.ProviderConfig{}, nil
}

func TestProductionServiceProcessesImageGenerationJobViaMockCapability(t *testing.T) {
	t.Parallel()

	ctx := testAuthCtx()
	productionRepo := repo.NewMemoryProductionRepository()
	productionService := NewProductionService(productionRepo, nil)
	productionService.SetProviderService(NewProviderService(&capabilityStubProviderRepo{capability: "image", providerType: "mock"}))

	jobID := "00000000-0000-0000-0000-000000000301"
	createdJob, err := productionRepo.CreateGenerationJob(ctx, repo.CreateGenerationJobParams{
		ID: jobID, ProjectID: "00000000-0000-0000-0000-000000000302",
		EpisodeID:  "00000000-0000-0000-0000-000000000303",
		RequestKey: "shot-image:test", Provider: "openai",
		Model: "gpt-image-1", TaskType: string(provider.TaskTypeImage),
		Status: domain.GenerationJobStatusQueued, Prompt: "cinematic neon alley",
		Params:       map[string]any{"width": 1024, "height": 1024},
		EventMessage: "shot image generation queued",
	})
	if err != nil {
		t.Fatalf("create generation job: %v", err)
	}

	summary, err := productionService.ProcessQueuedGenerationJobs(ctx, jobs.DefaultExecutionLimit)
	if err != nil {
		t.Fatalf("process image generation job: %v", err)
	}
	if summary.Processed != 1 || summary.Succeeded != 1 || summary.Failed != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	completed, err := productionService.GetGenerationJob(ctx, createdJob.ID)
	if err != nil {
		t.Fatalf("get completed image job: %v", err)
	}
	if completed.Status != domain.GenerationJobStatusSucceeded {
		t.Fatalf("expected succeeded image job, got %+v", completed)
	}
	if completed.ResultAssetID == "" {
		t.Fatalf("expected image job to persist a result asset id, got %+v", completed)
	}
	assets, err := productionService.ListEpisodeAssets(ctx, completed.EpisodeID)
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if len(assets) != 1 || assets[0].Kind != "image" || assets[0].Status != domain.AssetStatusReady {
		t.Fatalf("expected one ready image asset, got %+v", assets)
	}
}

func TestProductionServiceProcessesAudioGenerationJobViaMockCapability(t *testing.T) {
	t.Parallel()

	ctx := testAuthCtx()
	productionRepo := repo.NewMemoryProductionRepository()
	productionService := NewProductionService(productionRepo, nil)
	productionService.SetProviderService(NewProviderService(&capabilityStubProviderRepo{capability: "audio", providerType: "mock"}))

	jobID := "00000000-0000-0000-0000-000000000401"
	createdJob, err := productionRepo.CreateGenerationJob(ctx, repo.CreateGenerationJobParams{
		ID: jobID, ProjectID: "00000000-0000-0000-0000-000000000402",
		EpisodeID:  "00000000-0000-0000-0000-000000000403",
		RequestKey: "shot-audio:test", Provider: "openai",
		Model: "gpt-4o-mini-tts", TaskType: string(provider.TaskTypeTTS),
		Status: domain.GenerationJobStatusQueued, Prompt: "narrator says hello",
		Params:       map[string]any{"voice": "alloy", "format": "mp3"},
		EventMessage: "shot audio generation queued",
	})
	if err != nil {
		t.Fatalf("create generation job: %v", err)
	}

	summary, err := productionService.ProcessQueuedGenerationJobs(ctx, jobs.DefaultExecutionLimit)
	if err != nil {
		t.Fatalf("process audio generation job: %v", err)
	}
	if summary.Processed != 1 || summary.Succeeded != 1 || summary.Failed != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	completed, err := productionService.GetGenerationJob(ctx, createdJob.ID)
	if err != nil {
		t.Fatalf("get completed audio job: %v", err)
	}
	if completed.Status != domain.GenerationJobStatusSucceeded {
		t.Fatalf("expected succeeded audio job, got %+v", completed)
	}
	assets, err := productionService.ListEpisodeAssets(ctx, completed.EpisodeID)
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if len(assets) != 1 || assets[0].Kind != "audio" || assets[0].Status != domain.AssetStatusReady {
		t.Fatalf("expected one ready audio asset, got %+v", assets)
	}
}

func TestAudioResultURIPersistsBytesToMediaStorage(t *testing.T) {
	t.Parallel()

	storage := media.NewMemoryStorage()
	svc := NewProductionService(repo.NewMemoryProductionRepository(), nil)
	svc.SetMediaStorage(storage)

	job := domain.GenerationJob{ID: "job-audio-bytes", Params: map[string]any{"format": "mp3"}}
	uri, err := svc.audioResultURI(context.Background(), job, &provider.AudioResult{Bytes: []byte("RIFFfakeaudio")})
	if err != nil {
		t.Fatalf("audioResultURI returned error: %v", err)
	}
	if !strings.HasPrefix(uri, "mem://audio/") || !strings.HasSuffix(uri, ".mp3") {
		t.Fatalf("expected mem://audio/<job>.mp3 URI, got %q", uri)
	}
	rc, err := storage.Get(context.Background(), uri)
	if err != nil {
		t.Fatalf("storage.Get(%q) returned error: %v", uri, err)
	}
	defer rc.Close()
	buf := make([]byte, 64)
	n, _ := rc.Read(buf)
	if string(buf[:n]) != "RIFFfakeaudio" {
		t.Fatalf("expected stored bytes to round-trip, got %q", string(buf[:n]))
	}
}

func TestAudioResultURIFallsBackToInlineWhenStorageMissing(t *testing.T) {
	t.Parallel()

	svc := NewProductionService(repo.NewMemoryProductionRepository(), nil)
	uri, err := svc.audioResultURI(context.Background(), domain.GenerationJob{ID: "job-audio-fallback"}, &provider.AudioResult{Bytes: []byte("abcd")})
	if err != nil {
		t.Fatalf("audioResultURI returned error: %v", err)
	}
	if uri != "manmu://providers/audio/inline?bytes=4" {
		t.Fatalf("expected inline placeholder URI, got %q", uri)
	}
}

func TestAudioResultURIPrefersProviderURL(t *testing.T) {
	t.Parallel()

	storage := media.NewMemoryStorage()
	svc := NewProductionService(repo.NewMemoryProductionRepository(), nil)
	svc.SetMediaStorage(storage)
	uri, err := svc.audioResultURI(context.Background(), domain.GenerationJob{ID: "job-audio-url"}, &provider.AudioResult{URL: "https://cdn.example.com/a.mp3"})
	if err != nil {
		t.Fatalf("audioResultURI returned error: %v", err)
	}
	if uri != "https://cdn.example.com/a.mp3" {
		t.Fatalf("expected provider URL preserved, got %q", uri)
	}
}
