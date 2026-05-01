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
	AuthService       *service.AuthService
	ProjectService    *service.ProjectService
	ProductionService *service.ProductionService
	ProviderService   *service.ProviderService
	AgentService      *service.AgentService
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
	router.Get("/metrics", api.prometheusMetrics)

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(authContextMiddleware(cfg.AuthService))
		r.Get("/meta/capabilities", capabilitiesHandler(cfg.Version))
		r.Post("/auth/register", api.register)
		r.Post("/auth/login", api.login)
		r.Post("/auth/refresh", api.refreshSession)
		r.Post("/auth/logout", api.logoutSession)
		r.Get("/auth/me", api.currentSession)
		r.Get("/auth/sessions", api.listSessions)
		r.Post("/auth/sessions/{sessionId}:revoke", api.revokeSession)
		r.Get("/projects", api.listProjects)
		r.Post("/projects", api.createProject)
		r.Get("/projects/{projectId}", api.getProject)
		r.Get("/projects/{projectId}/episodes", api.listEpisodes)
		r.Post("/projects/{projectId}/episodes", api.createEpisode)
		r.Get("/episodes/{episodeId}", api.getEpisode)
		r.Get("/episodes/{episodeId}/story-sources", api.listStorySources)
		r.Post("/episodes/{episodeId}/story-sources", api.createStorySource)
		r.Post("/episodes/{episodeId}/story-analysis/start", api.startStoryAnalysis)
		r.Get("/episodes/{episodeId}/story-analyses", api.listStoryAnalyses)
		r.Get("/episodes/{episodeId}/approval-gates", api.listApprovalGates)
		r.Post("/episodes/{episodeId}/approval-gates:seed", api.seedApprovalGates)
		r.Post("/episodes/{episodeId}/production:seed", api.seedEpisodeProduction)
		r.Get("/episodes/{episodeId}/story-map", api.getStoryMap)
		r.Post("/episodes/{episodeId}/story-map:seed", api.seedStoryMap)
		r.Post("/story-map-characters/{characterId}/character-bible:save", api.saveCharacterBible)
		r.Get("/episodes/{episodeId}/storyboard-workspace", api.getStoryboardWorkspace)
		r.Get("/episodes/{episodeId}/storyboard-shots", api.listStoryboardShots)
		r.Post("/episodes/{episodeId}/storyboard-shots:seed", api.seedStoryboardShots)
		r.Post("/storyboard-shots/{shotId}:update", api.updateStoryboardShot)
		r.Get("/storyboard-shots/{shotId}/prompt-pack", api.getShotPromptPack)
		r.Get("/storyboard-shots/{shotId}/prompt-pack/recovery", api.getShotPromptPackRecovery)
		r.Post("/storyboard-shots/{shotId}/prompt-pack:generate", api.generateShotPromptPack)
		r.Post("/storyboard-shots/{shotId}/prompt-pack:save", api.saveShotPromptPack)
		r.Post("/storyboard-shots/{shotId}/videos:generate", api.startShotVideoGeneration)
		r.Get("/episodes/{episodeId}/assets", api.listEpisodeAssets)
		r.Post("/episodes/{episodeId}/assets:seed", api.seedEpisodeAssets)
		r.Post("/episodes/{episodeId}/timeline", api.saveEpisodeTimeline)
		r.Post("/episodes/{episodeId}/exports", api.startEpisodeExport)
		r.Post("/assets/{assetId}:lock", api.lockAsset)
		r.Get("/assets/{assetId}/recovery", api.getAssetRecovery)
		r.Post("/approval-gates/{gateId}:approve", api.approveApprovalGate)
		r.Post("/approval-gates/{gateId}:request-changes", api.requestApprovalChanges)
		r.Post("/approval-gates/{gateId}:resubmit", api.resubmitApprovalGate)
		r.Get("/story-analyses/{analysisId}", api.getStoryAnalysis)
		r.Get("/exports/{exportId}", api.getExport)
		r.Get("/exports/{exportId}/recovery", api.getExportRecovery)
		r.Get("/workflow-runs/{workflowRunId}", api.getWorkflowRun)
		r.Get("/generation-jobs", api.listGenerationJobs)
		r.Get("/generation-jobs/{jobId}", api.getGenerationJob)
		r.Get("/generation-jobs/{jobId}/recovery", api.getGenerationJobRecovery)
		r.Get("/episodes/{episodeId}/timeline", api.getEpisodeTimeline)
		r.Get("/events/stream", streamEventsHandler)
		r.Post("/agents/stream", api.streamAgentRun)

		// admin routes (owner/admin role required for reads; owner-only for provider mutations)
		r.Group(func(admin chi.Router) {
			admin.Use(requireRole("owner", "admin"))
			admin.Get("/admin/providers", api.listProviderConfigs)
			admin.Get("/admin/worker-metrics", api.getAdminWorkerMetrics)
			admin.Get("/admin/llm-telemetry", api.getAdminLLMTelemetry)
			admin.Post("/admin/llm-telemetry:reset", api.resetAdminLLMTelemetry)
			admin.Get("/admin/provider-audit", api.listProviderAuditEvents)
			admin.Post("/organizations/invitations", api.createInvitation)
			admin.Get("/organizations/invitations", api.listInvitations)
			admin.Get("/organizations/invitations/audit", api.listInvitationAudit)
			admin.Get("/organizations/invitations/audit/export", api.exportInvitationAudit)
			admin.Post("/organizations/invitations/{invitationId}:revoke", api.revokeInvitation)
			admin.Post("/organizations/invitations/{invitationId}:resend", api.resendInvitation)

			// owner-only mutations: provider config save / test 真实凭证写入
			admin.Group(func(owner chi.Router) {
				owner.Use(requireRole("owner"))
				owner.Post("/admin/providers:save", api.saveProviderConfig)
				owner.Post("/admin/providers/{capability}:test", api.testProviderConfig)
			})
		})
	})

	return router
}

type api struct {
	readinessChecker  Readiness
	authService       *service.AuthService
	projectService    *service.ProjectService
	productionService *service.ProductionService
	providerService   *service.ProviderService
	agentService      *service.AgentService
}

func newAPI(cfg RouterConfig) *api {
	return &api{
		readinessChecker:  cfg.Readiness,
		authService:       cfg.AuthService,
		projectService:    cfg.ProjectService,
		productionService: cfg.ProductionService,
		providerService:   cfg.ProviderService,
		agentService:      cfg.AgentService,
	}
}
