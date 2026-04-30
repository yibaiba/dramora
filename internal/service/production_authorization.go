package service

import (
	"context"
	"errors"
	"strings"

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
	if IsSystemAuthContext(ctx) {
		return nil
	}
	_, err := s.projectSvc.GetProject(ctx, projectID)
	return err
}

func (s *ProductionService) authorizeEpisode(ctx context.Context, episodeID string) error {
	if s.projectSvc == nil || episodeID == "" {
		return nil
	}
	if IsSystemAuthContext(ctx) {
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
	if IsSystemAuthContext(ctx) {
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

// workerJobAuthContextForProject 根据 project id 解析所属组织，并把当前 ctx
// 派生为带该组织上下文的 worker 身份。解析失败时返回原 ctx，让上层维持
// 既有 system 兜底语义。
func (s *ProductionService) workerJobAuthContextForProject(ctx context.Context, projectID string) context.Context {
	if s.projectSvc == nil || strings.TrimSpace(projectID) == "" {
		return ctx
	}
	project, err := s.projectSvc.LookupProjectByID(ctx, projectID)
	if err != nil || strings.TrimSpace(project.OrganizationID) == "" {
		return ctx
	}
	return WithRequestAuthContext(ctx, RequestAuthContext{
		OrganizationID: project.OrganizationID,
		Role:           RoleWorker,
	})
}

// workerJobAuthContextForTimeline 通过 timeline -> episode -> project 链路
// 解析 export 所属组织，并派生 worker auth context。
func (s *ProductionService) workerJobAuthContextForTimeline(ctx context.Context, timelineID string) context.Context {
	if s.projectSvc == nil || strings.TrimSpace(timelineID) == "" {
		return ctx
	}
	timeline, err := s.production.GetTimelineByID(ctx, timelineID)
	if err != nil || strings.TrimSpace(timeline.EpisodeID) == "" {
		return ctx
	}
	episode, err := s.projectSvc.LookupEpisodeByID(ctx, timeline.EpisodeID)
	if err != nil {
		return ctx
	}
	return s.workerJobAuthContextForProject(ctx, episode.ProjectID)
}
