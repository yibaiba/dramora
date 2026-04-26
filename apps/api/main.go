package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/yibaiba/dramora/internal/app"
	"github.com/yibaiba/dramora/internal/httpapi"
)

func main() {
	cfg, err := app.LoadConfig()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	container, err := app.NewContainer(context.Background(), cfg, logger)
	if err != nil {
		logger.Error("create container", "error", err)
		os.Exit(1)
	}
	defer container.Close()

	router := httpapi.NewRouter(httpapi.RouterConfig{
		Logger:            container.Logger,
		Version:           app.Version,
		Readiness:         container,
		ProjectService:    container.ProjectService,
		ProductionService: container.ProductionService,
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	if err := app.ListenAndServe(container.Context(), server, container.Logger, cfg.ShutdownTimeout); err != nil {
		container.Logger.Error("api server stopped", "error", err)
		os.Exit(1)
	}
}
