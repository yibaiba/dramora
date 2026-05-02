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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	container, err := app.NewContainer(ctx, cfg, logger)
	if err != nil {
		logger.Error("create container", "error", err)
		os.Exit(1)
	}
	defer container.Close()
	stopWorker := app.StartInlineWorker(ctx, cfg, container.Logger, container.ProductionService)
	defer stopWorker()

	router := httpapi.NewRouter(httpapi.RouterConfig{
		Logger:              container.Logger,
		Version:             app.Version,
		Readiness:           container,
		AuthService:         container.AuthService,
		ProjectService:      container.ProjectService,
		ProductionService:   container.ProductionService,
		ProviderService:     container.ProviderService,
		AgentService:        container.AgentService,
		WalletService:       container.WalletService,
		NotificationService: container.NotificationService,
		PaymentService:      container.PaymentService,
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
