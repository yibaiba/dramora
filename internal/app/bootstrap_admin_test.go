package app

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/yibaiba/dramora/internal/service"
)

func TestNewContainerBootstrapsLocalAdmin(t *testing.T) {
	isolateConfigEnv(t)
	t.Setenv("MANMU_DATA_DIR", t.TempDir())

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	container, err := NewContainer(context.Background(), cfg, logger)
	if err != nil {
		t.Fatalf("new container: %v", err)
	}
	defer container.Close()

	session, err := container.AuthService.Login(context.Background(), service.LoginInput{
		Email:    cfg.BootstrapAdminEmail,
		Password: cfg.BootstrapAdminPassword,
	})
	if err != nil {
		t.Fatalf("login bootstrap admin: %v", err)
	}
	if session.Role != "owner" {
		t.Fatalf("expected bootstrap admin owner role, got %q", session.Role)
	}
}
