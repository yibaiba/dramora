package app

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/yibaiba/dramora/internal/media"
	"github.com/yibaiba/dramora/internal/provider/payment"
	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/service"
)

type Container struct {
	cfg                  Config
	ctx                  context.Context
	db                   *repo.DB
	sqliteDB             *repo.SQLiteDB
	Logger               *slog.Logger
	AuthService          *service.AuthService
	ProjectService       *service.ProjectService
	ProductionService    *service.ProductionService
	ProviderService      *service.ProviderService
	AgentService         *service.AgentService
	WalletService        *service.WalletService
	NotificationService  *service.NotificationService
	PaymentService       *service.PaymentService
	PendingBillingWorker *service.PendingBillingWorker
	ReportService        *service.ReportService
}

func NewContainer(ctx context.Context, cfg Config, logger *slog.Logger) (*Container, error) {
	if logger == nil {
		logger = slog.Default()
	}

	projectRepo := repo.ProjectRepository(repo.NewMemoryProjectRepository())
	productionRepo := repo.ProductionRepository(repo.NewMemoryProductionRepository())
	identityRepo := repo.IdentityRepository(repo.NewMemoryIdentityRepository())
	refreshRepo := repo.RefreshTokenRepository(repo.NewMemoryRefreshTokenRepository())
	var db *repo.DB
	var sqliteDB *repo.SQLiteDB

	var providerService *service.ProviderService
	var workerMetricsRepo repo.WorkerMetricsRepository
	var llmTelemetryRepo repo.LLMTelemetryRepository
	var walletRepo repo.WalletRepository = repo.NewMemoryWalletRepository()
	var notificationRepo repo.NotificationRepository = repo.NewMemoryNotificationRepository()
	var pendingBillingRepo repo.PendingBillingRepository = repo.NewMemoryPendingBillingRepository()
	var paymentOrderRepo repo.PaymentOrderRepository = repo.NewMemoryPaymentOrderRepository()
	var billingReportRepo repo.BillingReportRepository = repo.NewMemoryBillingReportRepository()
	var operationCostRepo repo.OperationCostRepository = repo.NewMemoryOperationCostRepository()

	if cfg.DatabaseURL != "" {
		openedDB, err := repo.OpenPostgres(ctx, cfg.DatabaseURL)
		if err != nil {
			return nil, err
		}
		db = openedDB
		projectRepo = repo.NewPostgresProjectRepository(openedDB.Pool)
		productionRepo = repo.NewPostgresProductionRepository(openedDB.Pool)
		identityRepo = repo.NewPostgresIdentityRepository(openedDB.Pool)
		refreshRepo = repo.NewPostgresRefreshTokenRepository(openedDB.Pool)
		workerMetricsRepo = repo.NewPostgresWorkerMetricsRepository(openedDB.Pool)
		llmTelemetryRepo = repo.NewPostgresLLMTelemetryRepository(openedDB.Pool)
		walletRepo = repo.NewPostgresWalletRepository(openedDB.Pool)
		notificationRepo = repo.NewPostgresNotificationRepository(openedDB.Pool)
		pendingBillingRepo = repo.NewPostgresPendingBillingRepository(openedDB.Pool)
		paymentOrderRepo = repo.NewPaymentOrderRepository(openedDB.Pool)
		billingReportRepo = repo.NewPostgresBillingReportRepository(openedDB.Pool)
		operationCostRepo = repo.NewPostgresOperationCostRepository(openedDB.Pool)
	} else {
		dbPath := filepath.Join(cfg.DataDir, "data.db")
		openedDB, err := repo.OpenSQLite(ctx, dbPath)
		if err != nil {
			return nil, err
		}
		sqliteDB = openedDB
		projectRepo = repo.NewSQLiteProjectRepository(openedDB.DB)
		productionRepo = repo.NewSQLiteProductionRepository(openedDB.DB)
		identityRepo = repo.NewSQLiteIdentityRepository(openedDB.DB)
		refreshRepo = repo.NewSQLiteRefreshTokenRepository(openedDB.DB)
		providerService = service.NewProviderService(repo.NewSQLiteProviderConfigRepository(openedDB.DB))
		providerService.SetAuditRepository(repo.NewSQLiteProviderAuditRepository(openedDB.DB))
		workerMetricsRepo = repo.NewSQLiteWorkerMetricsRepository(openedDB.DB)
		llmTelemetryRepo = repo.NewSQLiteLLMTelemetryRepository(openedDB.DB)
		walletRepo = repo.NewSQLiteWalletRepository(openedDB.DB)
		notificationRepo = repo.NewSQLiteNotificationRepository(openedDB.DB)
		pendingBillingRepo = repo.NewSQLitePendingBillingRepository(openedDB.DB)
		paymentOrderRepo = repo.NewMemoryPaymentOrderRepository()
		logger.Info("using SQLite", "path", dbPath)
	}

	productionSvc := service.NewProductionService(productionRepo, nil)
	var agentSvc *service.AgentService
	if providerService != nil {
		agentSvc = service.NewAgentService(providerService)
		productionSvc.SetAgentService(agentSvc)
		productionSvc.SetProviderService(providerService)
		if llmTelemetryRepo != nil {
			agentSvc.SetTelemetryRepository(llmTelemetryRepo)
			if err := agentSvc.HydrateTelemetry(ctx); err != nil {
				logger.Warn("load llm telemetry failed", "err", err)
			}
		}
	}

	projectSvc := service.NewProjectService(projectRepo)
	productionSvc.SetProjectService(projectSvc)
	if cfg.MediaDir != "" {
		fsStore, err := media.NewFilesystemStorage(cfg.MediaDir)
		if err != nil {
			logger.Warn("filesystem media storage init failed, falling back to memory", "err", err, "dir", cfg.MediaDir)
			productionSvc.SetMediaStorage(media.NewMemoryStorage())
		} else {
			logger.Info("media storage", "backend", "filesystem", "root", fsStore.Root())
			productionSvc.SetMediaStorage(fsStore)
		}
	} else {
		productionSvc.SetMediaStorage(media.NewMemoryStorage())
	}

	if workerMetricsRepo != nil {
		productionSvc.SetWorkerMetricsRepository(workerMetricsRepo, logger)
		if err := productionSvc.LoadWorkerMetrics(ctx); err != nil {
			logger.Warn("load worker metrics failed", "err", err)
		}
	}

	notificationSvc := service.NewNotificationService(notificationRepo)
	if providerService != nil {
		providerService.SetNotificationService(notificationSvc)
	}
	authService := service.NewAuthService(identityRepo, cfg.JWTSecret, notificationSvc)
	authService.SetRefreshTokenRepository(refreshRepo)

	walletSvc := service.NewWalletService(walletRepo, notificationSvc)
	walletSvc.SetPendingBillingRepository(pendingBillingRepo)
	walletSvc.SetOperationCostRepository(operationCostRepo)

	pendingBillingWorker := service.NewPendingBillingWorker(logger, pendingBillingRepo, walletSvc)

	// 初始化支付提供商（Stripe）
	stripeProvider := payment.NewStripeProvider(
		cfg.StripeSecretKey,
		cfg.StripeWebhookSecret,
		cfg.StripeSuccessURL,
		cfg.StripeCancelURL,
	)
	paymentSvc := service.NewPaymentService(paymentOrderRepo, walletSvc, stripeProvider, logger)

	// 初始化报表服务
	reportSvc := service.NewReportService(walletRepo, pendingBillingRepo, operationCostRepo, billingReportRepo)

	return &Container{
		cfg:                  cfg,
		ctx:                  ctx,
		db:                   db,
		sqliteDB:             sqliteDB,
		Logger:               logger.With("env", cfg.Env),
		AuthService:          authService,
		ProjectService:       projectSvc,
		ProductionService:    productionSvc,
		ProviderService:      providerService,
		AgentService:         agentSvc,
		WalletService:        walletSvc,
		NotificationService:  notificationSvc,
		PaymentService:       paymentSvc,
		PendingBillingWorker: pendingBillingWorker,
		ReportService:        reportSvc,
	}, nil
}

func (c *Container) Config() Config {
	return c.cfg
}

func (c *Container) Context() context.Context {
	return c.ctx
}

func (c *Container) Ready(ctx context.Context) error {
	if c.db != nil {
		return c.db.Ready(ctx)
	}
	if c.sqliteDB != nil {
		return c.sqliteDB.Ready(ctx)
	}
	return nil
}

func (c *Container) Close() {
	if c.db != nil {
		c.db.Close()
	}
	if c.sqliteDB != nil {
		c.sqliteDB.Close()
	}
}
