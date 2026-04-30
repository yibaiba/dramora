package service

import (
	"context"
	"errors"
	"testing"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

const testOrganizationID = "00000000-0000-0000-0000-000000000001"

func testAuthCtx() context.Context {
	return WithRequestAuthContext(context.Background(), RequestAuthContext{
		OrganizationID: testOrganizationID,
		Role:           "owner",
	})
}

func TestProjectServiceCreatesProjectAndEpisode(t *testing.T) {
	t.Parallel()

	service := NewProjectService(repo.NewMemoryProjectRepository())
	ctx := testAuthCtx()

	project, err := service.CreateProject(ctx, CreateProjectInput{Name: " 漫幕测试 "})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if project.Name != "漫幕测试" {
		t.Fatalf("expected trimmed project name, got %q", project.Name)
	}

	episode, err := service.CreateEpisode(ctx, CreateEpisodeInput{
		ProjectID: project.ID,
		Title:     " 第一集 ",
	})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	if episode.Number != 1 {
		t.Fatalf("expected first episode number 1, got %d", episode.Number)
	}
	if episode.Title != "第一集" {
		t.Fatalf("expected trimmed title, got %q", episode.Title)
	}
}

func TestProjectServiceRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	service := NewProjectService(repo.NewMemoryProjectRepository())
	ctx := testAuthCtx()

	_, err := service.CreateProject(ctx, CreateProjectInput{Name: " "})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}

	_, err = service.GetProject(ctx, "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
