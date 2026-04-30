package service

import (
	"context"
	"testing"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/jobs"
	"github.com/yibaiba/dramora/internal/repo"
)

func TestProductionServiceWorkerMetricsRecordsOrgUnresolvedSkips(t *testing.T) {
	t.Parallel()

	ctx := testAuthCtx()
	productionRepo := repo.NewMemoryProductionRepository()
	productionService := NewProductionService(productionRepo, nil)
	// Inject a project service so worker_org_resolution actually runs (no-op
	// projectSvc would short-circuit and bypass the skip path).
	productionService.SetProjectService(NewProjectService(repo.NewMemoryProjectRepository()))

	// Seed a generation job whose project does not exist anywhere — so the
	// worker should fail to resolve the org and skip with a metric increment.
	jobID, err := domain.NewID()
	if err != nil {
		t.Fatalf("new id: %v", err)
	}
	if _, err := productionRepo.CreateGenerationJob(context.Background(), repo.CreateGenerationJobParams{
		ID:         jobID,
		ProjectID:  "00000000-0000-0000-0000-0000000000ff",
		EpisodeID:  "00000000-0000-0000-0000-0000000000fe",
		Provider:   "internal",
		TaskType:   "story-analysis",
		RequestKey: "metric-test:" + jobID,
		Status:     domain.GenerationJobStatusQueued,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	summary, err := productionService.ProcessQueuedGenerationJobs(ctx, jobs.DefaultExecutionLimit)
	if err != nil {
		t.Fatalf("process queued generation jobs: %v", err)
	}
	if summary.Processed != 0 || summary.Succeeded != 0 || summary.Failed != 0 {
		t.Fatalf("expected job to be skipped, got summary: %+v", summary)
	}

	snap := productionService.WorkerMetrics()
	if snap.GenerationOrgUnresolvedSkips != 1 {
		t.Fatalf("expected GenerationOrgUnresolvedSkips=1, got %d", snap.GenerationOrgUnresolvedSkips)
	}
	if snap.LastSkipKind != "generation" {
		t.Fatalf("expected LastSkipKind=generation, got %q", snap.LastSkipKind)
	}
	if snap.LastSkipReason == "" {
		t.Fatal("expected LastSkipReason to be populated")
	}
	if snap.LastSkipAt.IsZero() {
		t.Fatal("expected LastSkipAt to be populated")
	}
}
