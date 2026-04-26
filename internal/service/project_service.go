package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

type ProjectService struct {
	projects              repo.ProjectRepository
	defaultOrganizationID string
}

type CreateProjectInput struct {
	Name        string
	Description string
}

type CreateEpisodeInput struct {
	ProjectID string
	Number    int
	Title     string
}

func NewProjectService(projects repo.ProjectRepository, defaultOrganizationID string) *ProjectService {
	return &ProjectService{
		projects:              projects,
		defaultOrganizationID: defaultOrganizationID,
	}
}

func (s *ProjectService) ListProjects(ctx context.Context) ([]domain.Project, error) {
	return s.projects.ListProjects(ctx, s.defaultOrganizationID)
}

func (s *ProjectService) CreateProject(ctx context.Context, input CreateProjectInput) (domain.Project, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.Project{}, fmt.Errorf("%w: project name is required", domain.ErrInvalidInput)
	}

	id, err := domain.NewID()
	if err != nil {
		return domain.Project{}, err
	}

	return s.projects.CreateProject(ctx, repo.CreateProjectParams{
		ID:             id,
		OrganizationID: s.defaultOrganizationID,
		Name:           name,
		Description:    strings.TrimSpace(input.Description),
		Status:         domain.ProjectStatusDraft,
	})
}

func (s *ProjectService) GetProject(ctx context.Context, projectID string) (domain.Project, error) {
	if strings.TrimSpace(projectID) == "" {
		return domain.Project{}, fmt.Errorf("%w: project id is required", domain.ErrInvalidInput)
	}
	return s.projects.GetProject(ctx, s.defaultOrganizationID, projectID)
}

func (s *ProjectService) ListEpisodes(ctx context.Context, projectID string) ([]domain.Episode, error) {
	if _, err := s.GetProject(ctx, projectID); err != nil {
		return nil, err
	}
	return s.projects.ListEpisodes(ctx, projectID)
}

func (s *ProjectService) CreateEpisode(ctx context.Context, input CreateEpisodeInput) (domain.Episode, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return domain.Episode{}, fmt.Errorf("%w: episode title is required", domain.ErrInvalidInput)
	}
	if _, err := s.GetProject(ctx, input.ProjectID); err != nil {
		return domain.Episode{}, err
	}

	number, err := s.nextEpisodeNumber(ctx, input.ProjectID, input.Number)
	if err != nil {
		return domain.Episode{}, err
	}

	id, err := domain.NewID()
	if err != nil {
		return domain.Episode{}, err
	}

	return s.projects.CreateEpisode(ctx, repo.CreateEpisodeParams{
		ID:        id,
		ProjectID: input.ProjectID,
		Number:    number,
		Title:     title,
		Status:    domain.EpisodeStatusDraft,
	})
}

func (s *ProjectService) GetEpisode(ctx context.Context, episodeID string) (domain.Episode, error) {
	if strings.TrimSpace(episodeID) == "" {
		return domain.Episode{}, fmt.Errorf("%w: episode id is required", domain.ErrInvalidInput)
	}
	episode, err := s.projects.GetEpisode(ctx, episodeID)
	if err != nil {
		return domain.Episode{}, err
	}
	if _, err := s.GetProject(ctx, episode.ProjectID); err != nil {
		return domain.Episode{}, err
	}
	return episode, nil
}

func (s *ProjectService) nextEpisodeNumber(ctx context.Context, projectID string, requested int) (int, error) {
	if requested > 0 {
		return requested, nil
	}

	episodes, err := s.projects.ListEpisodes(ctx, projectID)
	if err != nil {
		return 0, err
	}
	return len(episodes) + 1, nil
}
