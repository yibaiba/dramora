package repo

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSQLiteIdentityRepositoryAuthIdentityRoundTrip(t *testing.T) {
	t.Parallel()

	sqliteDB, err := OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "identity.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer sqliteDB.Close()

	repo := NewSQLiteIdentityRepository(sqliteDB.DB)
	ctx := context.Background()
	if err := repo.CreateOrganization(ctx, CreateOrganizationParams{
		OrganizationID: "org-1",
		Name:           "Demo Workspace",
	}); err != nil {
		t.Fatalf("create organization: %v", err)
	}

	created, err := repo.CreateUserWithMembership(ctx, CreateUserWithMembershipParams{
		UserID:         "user-1",
		OrganizationID: "org-1",
		Email:          "demo@example.com",
		DisplayName:    "Demo User",
		PasswordHash:   "hash",
		Role:           "owner",
	})
	if err != nil {
		t.Fatalf("create user with membership: %v", err)
	}
	if created.User.CreatedAt.IsZero() || created.User.UpdatedAt.IsZero() {
		t.Fatalf("expected parsed timestamps, got created=%v updated=%v", created.User.CreatedAt, created.User.UpdatedAt)
	}

	lookedUp, err := repo.GetAuthIdentityByEmail(ctx, "demo@example.com")
	if err != nil {
		t.Fatalf("get auth identity by email: %v", err)
	}
	if lookedUp.User.ID != "user-1" || lookedUp.OrganizationID != "org-1" || lookedUp.Role != "owner" {
		t.Fatalf("unexpected auth identity: %+v", lookedUp)
	}
	if lookedUp.User.CreatedAt.IsZero() || lookedUp.User.UpdatedAt.IsZero() {
		t.Fatalf("expected parsed lookup timestamps, got created=%v updated=%v", lookedUp.User.CreatedAt, lookedUp.User.UpdatedAt)
	}
}
