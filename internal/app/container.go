package app

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/yibaiba/dramora/internal/media"
	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/service"
)

type Container struct {
	cfg               Config
	ctx               context.Context
	db                *repo.DB
	sqliteDB          *repo.SQLiteDB
	Logger            *slog.Logger
	AuthService       *service.AuthService
	ProjectService    *service.ProjectService
	ProductionService *service.ProductionService
	ProviderService   *service.ProviderService
	AgentService      *service.AgentService
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
		logger.Info("using SQLite", "path", dbPath)
	}

	productionSvc := service.NewProductionService(productionRepo, nil)
	var agentSvc *service.AgentService
	if providerService != nil {
		agentSvc = service.NewAgentService(providerService)
		productionSvc.SetAgentService(agentSvc)
		productionSvc.SetProviderService(providerService)
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

	authService := service.NewAuthService(identityRepo, cfg.JWTSecret)
	authService.SetRefreshTokenRepository(refreshRepo)

	return &Container{
		cfg:               cfg,
		ctx:               ctx,
		db:                db,
		sqliteDB:          sqliteDB,
		Logger:            logger.With("env", cfg.Env),
		AuthService:       authService,
		ProjectService:    projectSvc,
		ProductionService: productionSvc,
		ProviderService:   providerService,
		AgentService:      agentSvc,
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
