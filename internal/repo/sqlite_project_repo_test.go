package repo

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/yibaiba/dramora/internal/domain"
)

func TestSQLiteProjectRepositoryProjectAndEpisodeRoundTrip(t *testing.T) {
	t.Parallel()

	sqliteDB, err := OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "project.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer sqliteDB.Close()

	ctx := context.Background()
	identityRepo := NewSQLiteIdentityRepository(sqliteDB.DB)
	if err := identityRepo.CreateOrganization(ctx, CreateOrganizationParams{
		OrganizationID: "org-1",
		Name:           "Demo Workspace",
	}); err != nil {
		t.Fatalf("create organization: %v", err)
	}

	projectRepo := NewSQLiteProjectRepository(sqliteDB.DB)
	project, err := projectRepo.CreateProject(ctx, CreateProjectParams{
		ID:             "project-1",
		OrganizationID: "org-1",
		Name:           "联调项目",
		Description:    "sqlite round trip",
		Status:         domain.ProjectStatusDraft,
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if project.CreatedAt.IsZero() || project.UpdatedAt.IsZero() {
		t.Fatalf("expected parsed project timestamps, got created=%v updated=%v", project.CreatedAt, project.UpdatedAt)
	}

	projects, err := projectRepo.ListProjects(ctx, "org-1")
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if len(projects) != 1 || projects[0].ID != "project-1" {
		t.Fatalf("unexpected projects: %+v", projects)
	}

	episode, err := projectRepo.CreateEpisode(ctx, CreateEpisodeParams{
		ID:        "episode-1",
		ProjectID: "project-1",
		Number:    1,
		Title:     "第一集",
		Status:    domain.EpisodeStatusDraft,
	})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	if episode.CreatedAt.IsZero() || episode.UpdatedAt.IsZero() {
		t.Fatalf("expected parsed episode timestamps, got created=%v updated=%v", episode.CreatedAt, episode.UpdatedAt)
	}

	episodes, err := projectRepo.ListEpisodes(ctx, "project-1")
	if err != nil {
		t.Fatalf("list episodes: %v", err)
	}
	if len(episodes) != 1 || episodes[0].ID != "episode-1" {
		t.Fatalf("unexpected episodes: %+v", episodes)
	}
}
