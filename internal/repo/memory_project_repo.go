package repo

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
)

type MemoryProjectRepository struct {
	mu       sync.RWMutex
	projects map[string]domain.Project
	episodes map[string]domain.Episode
}

func NewMemoryProjectRepository() *MemoryProjectRepository {
	return &MemoryProjectRepository{
		projects: make(map[string]domain.Project),
		episodes: make(map[string]domain.Episode),
	}
}

func (r *MemoryProjectRepository) ListProjects(
	_ context.Context,
	organizationID string,
) ([]domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projects := make([]domain.Project, 0, len(r.projects))
	for _, project := range r.projects {
		if project.OrganizationID == organizationID {
			projects = append(projects, project)
		}
	}
	sort.Slice(projects, func(i int, j int) bool {
		return projects[i].CreatedAt.After(projects[j].CreatedAt)
	})
	return projects, nil
}

func (r *MemoryProjectRepository) CreateProject(
	_ context.Context,
	params CreateProjectParams,
) (domain.Project, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	project := domain.Project{
		ID:             params.ID,
		OrganizationID: params.OrganizationID,
		Name:           params.Name,
		Description:    params.Description,
		Status:         params.Status,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	r.projects[project.ID] = project
	return project, nil
}

func (r *MemoryProjectRepository) GetProject(
	_ context.Context,
	organizationID string,
	projectID string,
) (domain.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	project, ok := r.projects[projectID]
	if !ok || project.OrganizationID != organizationID {
		return domain.Project{}, domain.ErrNotFound
	}
	return project, nil
}

func (r *MemoryProjectRepository) ListEpisodes(_ context.Context, projectID string) ([]domain.Episode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	episodes := make([]domain.Episode, 0)
	for _, episode := range r.episodes {
		if episode.ProjectID == projectID {
			episodes = append(episodes, episode)
		}
	}
	sort.Slice(episodes, func(i int, j int) bool {
		return episodes[i].Number < episodes[j].Number
	})
	return episodes, nil
}

func (r *MemoryProjectRepository) CreateEpisode(
	_ context.Context,
	params CreateEpisodeParams,
) (domain.Episode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.projects[params.ProjectID]; !ok {
		return domain.Episode{}, domain.ErrNotFound
	}
	for _, existing := range r.episodes {
		if existing.ProjectID == params.ProjectID && existing.Number == params.Number {
			return domain.Episode{}, domain.ErrInvalidInput
		}
	}

	now := time.Now().UTC()
	episode := domain.Episode{
		ID:        params.ID,
		ProjectID: params.ProjectID,
		Number:    params.Number,
		Title:     params.Title,
		Status:    params.Status,
		CreatedAt: now,
		UpdatedAt: now,
	}
	r.episodes[episode.ID] = episode
	return episode, nil
}

func (r *MemoryProjectRepository) GetEpisode(_ context.Context, episodeID string) (domain.Episode, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	episode, ok := r.episodes[episodeID]
	if !ok {
		return domain.Episode{}, domain.ErrNotFound
	}
	return episode, nil
}
