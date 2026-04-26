package app

import (
	"context"
	"log/slog"

	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/service"
)

type Container struct {
	cfg               Config
	ctx               context.Context
	db                *repo.DB
	Logger            *slog.Logger
	ProjectService    *service.ProjectService
	ProductionService *service.ProductionService
}

func NewContainer(ctx context.Context, cfg Config, logger *slog.Logger) (*Container, error) {
	if logger == nil {
		logger = slog.Default()
	}

	projectRepo := repo.ProjectRepository(repo.NewMemoryProjectRepository())
	productionRepo := repo.ProductionRepository(repo.NewMemoryProductionRepository())
	var db *repo.DB
	if cfg.DatabaseURL != "" {
		openedDB, err := repo.OpenPostgres(ctx, cfg.DatabaseURL)
		if err != nil {
			return nil, err
		}
		db = openedDB
		projectRepo = repo.NewPostgresProjectRepository(openedDB.Pool)
		productionRepo = repo.NewPostgresProductionRepository(openedDB.Pool)
	}

	return &Container{
		cfg:               cfg,
		ctx:               ctx,
		db:                db,
		Logger:            logger.With("env", cfg.Env),
		ProjectService:    service.NewProjectService(projectRepo, cfg.DefaultOrganizationID),
		ProductionService: service.NewProductionService(productionRepo, nil),
	}, nil
}

func (c *Container) Config() Config {
	return c.cfg
}

func (c *Container) Context() context.Context {
	return c.ctx
}

func (c *Container) Ready(ctx context.Context) error {
	if c.db == nil {
		return nil
	}
	return c.db.Ready(ctx)
}

func (c *Container) Close() {
	if c.db != nil {
		c.db.Close()
	}
}
