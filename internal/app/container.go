package app

import (
	"context"
	"log/slog"
	"path/filepath"

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
}

func NewContainer(ctx context.Context, cfg Config, logger *slog.Logger) (*Container, error) {
	if logger == nil {
		logger = slog.Default()
	}

	projectRepo := repo.ProjectRepository(repo.NewMemoryProjectRepository())
	productionRepo := repo.ProductionRepository(repo.NewMemoryProductionRepository())
	identityRepo := repo.IdentityRepository(repo.NewMemoryIdentityRepository())
	var db *repo.DB
	var sqliteDB *repo.SQLiteDB

	var providerService *service.ProviderService

	if cfg.DatabaseURL != "" {
		openedDB, err := repo.OpenPostgres(ctx, cfg.DatabaseURL)
		if err != nil {
			return nil, err
		}
		db = openedDB
		projectRepo = repo.NewPostgresProjectRepository(openedDB.Pool)
		productionRepo = repo.NewPostgresProductionRepository(openedDB.Pool)
		identityRepo = repo.NewPostgresIdentityRepository(openedDB.Pool)
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
		providerService = service.NewProviderService(repo.NewSQLiteProviderConfigRepository(openedDB.DB))
		logger.Info("using SQLite", "path", dbPath)
	}

	productionSvc := service.NewProductionService(productionRepo, nil)
	if providerService != nil {
		productionSvc.SetAgentService(service.NewAgentService(providerService))
	}

	projectSvc := service.NewProjectService(projectRepo, cfg.DefaultOrganizationID)
	productionSvc.SetProjectService(projectSvc)

	return &Container{
		cfg:               cfg,
		ctx:               ctx,
		db:                db,
		sqliteDB:          sqliteDB,
		Logger:            logger.With("env", cfg.Env),
		AuthService:       service.NewAuthService(identityRepo, cfg.JWTSecret),
		ProjectService:    projectSvc,
		ProductionService: productionSvc,
		ProviderService:   providerService,
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
