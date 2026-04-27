package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/yibaiba/dramora/internal/service"
)

type Readiness interface {
	Ready(ctx context.Context) error
}

type RouterConfig struct {
	Logger            *slog.Logger
	Version           string
	Readiness         Readiness
	ProjectService    *service.ProjectService
	ProductionService *service.ProductionService
}

func NewRouter(cfg RouterConfig) http.Handler {
	api := newAPI(cfg)
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(requestLogger(cfg.Logger))

	router.Get("/healthz", api.health)
	router.Get("/readyz", api.readiness)

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/meta/capabilities", capabilitiesHandler(cfg.Version))
		r.Get("/projects", api.listProjects)
		r.Post("/projects", api.createProject)
		r.Get("/projects/{projectId}", api.getProject)
		r.Get("/projects/{projectId}/episodes", api.listEpisodes)
		r.Post("/projects/{projectId}/episodes", api.createEpisode)
		r.Get("/episodes/{episodeId}", api.getEpisode)
		r.Post("/episodes/{episodeId}/story-analysis/start", api.startStoryAnalysis)
		r.Get("/episodes/{episodeId}/story-analyses", api.listStoryAnalyses)
		r.Get("/episodes/{episodeId}/approval-gates", api.listApprovalGates)
		r.Post("/episodes/{episodeId}/approval-gates:seed", api.seedApprovalGates)
		r.Get("/episodes/{episodeId}/story-map", api.getStoryMap)
		r.Post("/episodes/{episodeId}/story-map:seed", api.seedStoryMap)
		r.Get("/episodes/{episodeId}/storyboard-shots", api.listStoryboardShots)
		r.Post("/episodes/{episodeId}/storyboard-shots:seed", api.seedStoryboardShots)
		r.Post("/storyboard-shots/{shotId}:update", api.updateStoryboardShot)
		r.Get("/storyboard-shots/{shotId}/prompt-pack", api.getShotPromptPack)
		r.Post("/storyboard-shots/{shotId}/prompt-pack:generate", api.generateShotPromptPack)
		r.Post("/storyboard-shots/{shotId}/prompt-pack:save", api.saveShotPromptPack)
		r.Post("/storyboard-shots/{shotId}/videos:generate", api.startShotVideoGeneration)
		r.Get("/episodes/{episodeId}/assets", api.listEpisodeAssets)
		r.Post("/episodes/{episodeId}/assets:seed", api.seedEpisodeAssets)
		r.Post("/episodes/{episodeId}/timeline", api.saveEpisodeTimeline)
		r.Post("/episodes/{episodeId}/exports", api.startEpisodeExport)
		r.Post("/assets/{assetId}:lock", api.lockAsset)
		r.Post("/approval-gates/{gateId}:approve", api.approveApprovalGate)
		r.Post("/approval-gates/{gateId}:request-changes", api.requestApprovalChanges)
		r.Get("/story-analyses/{analysisId}", api.getStoryAnalysis)
		r.Get("/exports/{exportId}", api.getExport)
		r.Get("/workflow-runs/{workflowRunId}", api.getWorkflowRun)
		r.Get("/generation-jobs", api.listGenerationJobs)
		r.Get("/generation-jobs/{jobId}", api.getGenerationJob)
		r.Get("/episodes/{episodeId}/timeline", api.getEpisodeTimeline)
		r.Get("/events/stream", streamEventsHandler)
	})

	return router
}

type api struct {
	readinessChecker  Readiness
	projectService    *service.ProjectService
	productionService *service.ProductionService
}

func newAPI(cfg RouterConfig) *api {
	return &api{
		readinessChecker:  cfg.Readiness,
		projectService:    cfg.ProjectService,
		productionService: cfg.ProductionService,
	}
}
