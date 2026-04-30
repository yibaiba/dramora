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

func TestProductionServicePersistsAndReloadsWorkerMetrics(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	metricsRepo := repo.NewMemoryWorkerMetricsRepository()

	// First service instance: simulate a process that records skips and writes
	// them through to the persistent store.
	first := NewProductionService(repo.NewMemoryProductionRepository(), nil)
	first.SetWorkerMetricsRepository(metricsRepo, nil)

	first.metrics.recordGenerationSkip("project lookup failed")
	first.metrics.recordGenerationSkip("project lookup failed")
	first.metrics.recordExportSkip("timeline lookup failed")

	rows, err := metricsRepo.LoadAll(ctx)
	if err != nil {
		t.Fatalf("load all: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 metric rows persisted, got %d", len(rows))
	}

	// Second service instance: simulate a process restart that loads the prior
	// state from persistence and resumes counters.
	second := NewProductionService(repo.NewMemoryProductionRepository(), nil)
	second.SetWorkerMetricsRepository(metricsRepo, nil)
	if err := second.LoadWorkerMetrics(ctx); err != nil {
		t.Fatalf("load worker metrics: %v", err)
	}

	snap := second.WorkerMetrics()
	if snap.GenerationOrgUnresolvedSkips != 2 {
		t.Fatalf("expected GenerationOrgUnresolvedSkips=2 after reload, got %d", snap.GenerationOrgUnresolvedSkips)
	}
	if snap.ExportOrgUnresolvedSkips != 1 {
		t.Fatalf("expected ExportOrgUnresolvedSkips=1 after reload, got %d", snap.ExportOrgUnresolvedSkips)
	}
	if snap.LastSkipKind == "" || snap.LastSkipAt.IsZero() {
		t.Fatalf("expected last skip metadata to be restored, got %+v", snap)
	}
}

func TestProductionServiceWorkerMetricsAggregatedAcrossProcesses(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	shared := repo.NewMemoryWorkerMetricsRepository()

	// Two independent ProductionService instances share one persistent
	// metrics store, simulating two worker processes writing through to the
	// same backing table.
	procA := NewProductionService(repo.NewMemoryProductionRepository(), nil)
	procA.SetWorkerMetricsRepository(shared, nil)
	procB := NewProductionService(repo.NewMemoryProductionRepository(), nil)
	procB.SetWorkerMetricsRepository(shared, nil)

	procA.metrics.recordGenerationSkip("project lookup failed")
	procB.metrics.recordGenerationSkip("project lookup failed")
	procB.metrics.recordExportSkip("timeline lookup failed")

	// Local snapshots only know what each process counted.
	if local := procA.WorkerMetrics(); local.GenerationOrgUnresolvedSkips != 1 || local.Source != "local" {
		t.Fatalf("procA local snapshot mismatch: %+v", local)
	}
	if local := procB.WorkerMetrics(); local.GenerationOrgUnresolvedSkips != 1 || local.ExportOrgUnresolvedSkips != 1 {
		t.Fatalf("procB local snapshot mismatch: %+v", local)
	}

	// Aggregated snapshot reads the shared store, so both processes see the
	// cross-process totals (2 generation skips, 1 export skip).
	agg := procA.WorkerMetricsAggregated(ctx)
	if agg.Source != "aggregated" {
		t.Fatalf("expected Source=aggregated, got %q", agg.Source)
	}
	if agg.GenerationOrgUnresolvedSkips != 2 {
		t.Fatalf("expected aggregated GenerationOrgUnresolvedSkips=2, got %d", agg.GenerationOrgUnresolvedSkips)
	}
	if agg.ExportOrgUnresolvedSkips != 1 {
		t.Fatalf("expected aggregated ExportOrgUnresolvedSkips=1, got %d", agg.ExportOrgUnresolvedSkips)
	}
	if agg.LastSkipKind == "" || agg.LastSkipAt.IsZero() {
		t.Fatalf("expected aggregated last skip metadata, got %+v", agg)
	}
}

func TestProductionServiceWorkerMetricsAggregatedFallsBackWithoutRepo(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := NewProductionService(repo.NewMemoryProductionRepository(), nil)
	svc.metrics.recordGenerationSkip("local-only")

	snap := svc.WorkerMetricsAggregated(ctx)
	if snap.Source != "local" {
		t.Fatalf("expected fallback Source=local, got %q", snap.Source)
	}
	if snap.GenerationOrgUnresolvedSkips != 1 {
		t.Fatalf("expected local fallback counter=1, got %d", snap.GenerationOrgUnresolvedSkips)
	}
}
