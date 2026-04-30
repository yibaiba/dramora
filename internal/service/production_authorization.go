package service

import (
	"context"
	"errors"

	"github.com/yibaiba/dramora/internal/domain"
)

func (s *ProductionService) authorizeScopedResource(ctx context.Context, projectID string, episodeID string) error {
	if episodeID != "" {
		return s.authorizeEpisode(ctx, episodeID)
	}
	if projectID != "" {
		return s.authorizeProject(ctx, projectID)
	}
	return nil
}

func (s *ProductionService) authorizeProject(ctx context.Context, projectID string) error {
	if s.projectSvc == nil || projectID == "" {
		return nil
	}
	_, err := s.projectSvc.GetProject(ctx, projectID)
	return err
}

func (s *ProductionService) authorizeEpisode(ctx context.Context, episodeID string) error {
	if s.projectSvc == nil || episodeID == "" {
		return nil
	}
	_, err := s.projectSvc.GetEpisode(ctx, episodeID)
	return err
}

func (s *ProductionService) filterGenerationJobsForContext(
	ctx context.Context,
	jobs []domain.GenerationJob,
) ([]domain.GenerationJob, error) {
	if s.projectSvc == nil {
		return jobs, nil
	}

	authorizedProjects := make(map[string]error)
	filtered := make([]domain.GenerationJob, 0, len(jobs))
	for _, job := range jobs {
		projectID := job.ProjectID
		authErr, ok := authorizedProjects[projectID]
		if !ok {
			authErr = s.authorizeScopedResource(ctx, job.ProjectID, job.EpisodeID)
			authorizedProjects[projectID] = authErr
		}
		switch {
		case authErr == nil:
			filtered = append(filtered, job)
		case errors.Is(authErr, domain.ErrNotFound):
			continue
		default:
			return nil, authErr
		}
	}
	return filtered, nil
}
