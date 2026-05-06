package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/service"
)

func ensureLocalBootstrapAdmin(
	ctx context.Context,
	cfg Config,
	logger *slog.Logger,
	identityRepo repo.IdentityRepository,
	authService *service.AuthService,
) error {
	if !cfg.BootstrapAdminEnabled {
		return nil
	}

	_, err := identityRepo.GetAuthIdentityByEmail(ctx, cfg.BootstrapAdminEmail)
	switch {
	case err == nil:
		logger.Info("local bootstrap admin ready", "email", cfg.BootstrapAdminEmail)
		return nil
	case !errors.Is(err, domain.ErrNotFound):
		return fmt.Errorf("lookup local bootstrap admin: %w", err)
	}

	session, err := authService.Register(ctx, service.RegisterInput{
		Email:       cfg.BootstrapAdminEmail,
		DisplayName: cfg.BootstrapAdminDisplayName,
		Password:    cfg.BootstrapAdminPassword,
	})
	if err != nil {
		return fmt.Errorf("register local bootstrap admin: %w", err)
	}
	logger.Info("bootstrapped local admin", "email", session.User.Email, "organization_id", session.OrganizationID)
	return nil
}
