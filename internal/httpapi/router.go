package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/yibaiba/dramora/internal/jobs"
	"github.com/yibaiba/dramora/internal/realtime"
	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/service"
)

type Readiness interface {
	Ready(ctx context.Context) error
}

type RouterConfig struct {
	Logger                       *slog.Logger
	Version                      string
	Readiness                    Readiness
	AuthService                  *service.AuthService
	ProjectService               *service.ProjectService
	ProductionService            *service.ProductionService
	ProviderService              *service.ProviderService
	AgentService                 *service.AgentService
	WalletService                *service.WalletService
	NotificationService          *service.NotificationService
	PaymentService               *service.PaymentService
	ReportService                *service.ReportService
	RedemptionCodeService        *service.RedemptionCodeService
	BatchSubmissionService       *service.BatchSubmissionService
	ShortCodeRepository          repo.ShortCodeRepository
	ShortVideoTemplateRepository repo.ShortVideoTemplateRepository
	ShortVideoRepository         repo.ShortVideoRepository
	JobsClient                   jobs.Client
	WebSocketManager             *WebSocketManager
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
	router.Post("/webhook/payment", api.handlePaymentWebhook)
	router.Get("/r/:shortCode", api.redirectShortCode)

	// WebSocket route (requires auth)
	router.Route("/ws", func(r chi.Router) {
		r.Use(authContextMiddleware(cfg.AuthService))
		wsHandler := NewWebSocketHandler(cfg.WebSocketManager, cfg.Logger)
		r.Get("/", wsHandler.HandleWebSocket)
	})

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
		r.Post("/account:change-password", api.changeAccountPassword)
		r.Get("/account/api-keys", api.listAccountAPIKeys)
		r.Post("/account/api-keys", api.createAccountAPIKey)
		r.Post("/account/api-keys/{keyId}:update", api.updateAccountAPIKey)
		r.Post("/account/api-keys/{keyId}:toggle", api.toggleAccountAPIKey)
		r.Post("/account/api-keys/{keyId}:delete", api.deleteAccountAPIKey)
		r.Get("/workspaces", api.listWorkspaces)
		r.Get("/projects", api.listProjects)
		r.Post("/projects", api.createProject)
		r.Get("/projects/{projectId}", api.getProject)
		r.Get("/projects/{projectId}/episodes", api.listEpisodes)
		r.Post("/projects/{projectId}/episodes", api.createEpisode)
		r.Get("/episodes/{episodeId}", api.getEpisode)
		r.Post("/episodes/{episodeId}/chat", api.handleChatMessage)
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
		r.Post("/episodes/{episodeId}/batch-generate", api.batchGenerateShots)
		r.Post("/generation-jobs/{jobId}:retry", api.retryGenerationJob)
		r.Post("/generation-jobs/{jobId}:priority", api.updateGenerationJobPriority)
		r.Post("/episodes/{episodeId}/queue:pause", api.pauseEpisodeQueue)
		r.Post("/episodes/{episodeId}/queue:resume", api.resumeEpisodeQueue)
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
		r.Get("/agent-runs/{runId}", api.getAgentRun)
		r.Get("/agent-runs/{runId}/stream", api.reconnectAgentRunStream)
		r.Get("/wallet", api.getWallet)
		r.Get("/wallet/transactions", api.listWalletTransactions)
		r.Post("/wallet:charge", api.handleChargeWallet)
		r.Post("/wallet:charge:initiate", api.initiateChargeWallet)
		r.Get("/operation-costs", api.getOperationCosts)
		r.Get("/notifications", api.listNotifications)
		r.Post("/redemption-codes:redeem", api.redeemCode)

		// Short video routes
		svHandler := NewShortVideoHandler(api.shortVideoTemplateRepo, api.shortVideoRepo, api.productionService)
		svHandler.RegisterRoutes(r)

		// Batch submission routes
		batchHandler := NewBatchSubmissionHandler(api.batchSubmissionService)
		r.Post("/batch-submissions:create", batchHandler.CreateBatchSubmission)
		r.Get("/batch-submissions", batchHandler.ListBatchSubmissions)
		r.Get("/batch-submissions/{id}", batchHandler.GetBatchSubmission)
		r.Post("/batch-submissions/{id}:cancel", batchHandler.CancelBatchSubmission)
		r.Post("/batch-submissions/{id}/videos/{videoId}:retry", batchHandler.RetryVideo)

		// admin routes (owner/admin role required for reads; owner-only for provider mutations)
		r.Group(func(admin chi.Router) {
			admin.Use(requireRole("owner", "admin"))
			admin.Post("/notifications", api.createNotification)
			admin.Get("/admin/providers", api.listProviderConfigs)
			admin.Get("/admin/worker-metrics", api.getAdminWorkerMetrics)
			admin.Get("/admin/llm-telemetry", api.getAdminLLMTelemetry)
			admin.Post("/admin/llm-telemetry:reset", api.resetAdminLLMTelemetry)
			admin.Get("/admin/provider-audit", api.listProviderAuditEvents)
			admin.Get("/admin/operation-costs", api.getAdminOperationCosts)
			admin.Post("/admin/operation-costs:update", api.updateAdminOperationCosts)
			admin.Get("/admin/operation-costs/{operationType}/history", api.getAdminOperationCostHistory)
			admin.Get("/admin/billing-reports", api.getAdminBillingReports)
			admin.Post("/admin/billing-reports:generate", api.generateAdminBillingReport)
			admin.Get("/admin/billing-reports/{reportID}", api.getAdminBillingReportByID)
			admin.Get("/admin/billing-reports/{reportID}/summary", api.getAdminBillingReportSummary)
			admin.Get("/organizations/members", api.listOrganizationMembers)
			admin.Post("/organizations/members/{userId}/role", api.updateOrganizationMemberRole)
			admin.Post("/organizations/members/{userId}:remove", api.removeOrganizationMember)
			admin.Post("/organizations/invitations", api.createInvitation)
			admin.Get("/organizations/invitations", api.listInvitations)
			admin.Get("/organizations/invitations/audit", api.listInvitationAudit)
			admin.Get("/organizations/invitations/audit/export", api.exportInvitationAudit)
			admin.Post("/organizations/invitations/{invitationId}:revoke", api.revokeInvitation)
			admin.Post("/organizations/invitations/{invitationId}:resend", api.resendInvitation)
			admin.Post("/wallet:credit", api.creditWallet)
			admin.Post("/wallet:debit", api.debitWallet)
			admin.Post("/wallet/preview-cost", api.previewWalletCost)
			admin.Post("/notifications/{id}:read", api.markNotificationAsRead)
			admin.Post("/notifications:read-all", api.markAllNotificationsAsRead)
			admin.Post("/admin/redemption-codes:generate", api.generateCodes)
			admin.Post("/admin/redemption-campaigns:create", api.createCampaign)
			admin.Get("/admin/redemption-campaigns/{campaignId}/stats", api.getCampaignStats)
			admin.Post("/admin/redemption-codes:send-email", api.sendEmail)
			admin.Post("/redemption-codes:shorten", api.shortenCode)

			// owner-only mutations: provider config save / test 真实凭证写入
			admin.Group(func(owner chi.Router) {
				owner.Use(requireRole("owner"))
				owner.Post("/admin/providers:save", api.saveProviderConfig)
				owner.Post("/admin/providers/{capability}:test", api.testProviderConfig)
				owner.Post("/admin/providers/chat:smoke", api.smokeChatProvider)
				owner.Post("/admin/providers/chat:smoke-stream", api.smokeChatProviderStream)
			})
		})
	})

	return router
}

type api struct {
	readinessChecker       Readiness
	logger                 *slog.Logger
	authService            *service.AuthService
	projectService         *service.ProjectService
	productionService      *service.ProductionService
	providerService        *service.ProviderService
	agentService           *service.AgentService
	agentRunManager        *realtime.AgentRunManager
	walletService          *service.WalletService
	notificationService    *service.NotificationService
	paymentService         *service.PaymentService
	reportService          *service.ReportService
	redemptionCodeService  *service.RedemptionCodeService
	batchSubmissionService *service.BatchSubmissionService
	shortCodeRepo          repo.ShortCodeRepository
	shortVideoTemplateRepo repo.ShortVideoTemplateRepository
	shortVideoRepo         repo.ShortVideoRepository
	jobsClient             jobs.Client
}

func newAPI(cfg RouterConfig) *api {
	return &api{
		readinessChecker:       cfg.Readiness,
		logger:                 cfg.Logger,
		authService:            cfg.AuthService,
		projectService:         cfg.ProjectService,
		productionService:      cfg.ProductionService,
		providerService:        cfg.ProviderService,
		agentService:           cfg.AgentService,
		agentRunManager:        realtime.NewAgentRunManager(),
		walletService:          cfg.WalletService,
		notificationService:    cfg.NotificationService,
		paymentService:         cfg.PaymentService,
		reportService:          cfg.ReportService,
		redemptionCodeService:  cfg.RedemptionCodeService,
		batchSubmissionService: cfg.BatchSubmissionService,
		shortCodeRepo:          cfg.ShortCodeRepository,
		shortVideoTemplateRepo: cfg.ShortVideoTemplateRepository,
		shortVideoRepo:         cfg.ShortVideoRepository,
		jobsClient:             cfg.JobsClient,
	}
}
