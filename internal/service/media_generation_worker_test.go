package service

import (
	"context"
	"testing"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/jobs"
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
