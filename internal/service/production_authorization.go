package service

import (
	"context"
	"errors"
	"fmt"
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

// workerJobAuthContextForProject 根据 project id 解析所属组织，并把当前 ctx
// 派生为带该组织上下文的 worker 身份。
//
// 行为契约：
//   - projectSvc 未注入（多见于单元测试）时返回 (ctx, nil)，由 authorize* 自身的
//     "projectSvc == nil 即放行" 兜底。
//   - 真实运行环境下若 lookup 失败或组织缺失，返回 (ctx, error)，由 worker 调用方
//     跳过该 job 并记录日志，不再静默回退到 system bypass。
func (s *ProductionService) workerJobAuthContextForProject(ctx context.Context, projectID string) (context.Context, error) {
	if s.projectSvc == nil {
		return ctx, nil
	}
	if strings.TrimSpace(projectID) == "" {
		return ctx, fmt.Errorf("%w: worker job missing project id", domain.ErrInvalidInput)
	}
	project, err := s.projectSvc.LookupProjectByID(ctx, projectID)
	if err != nil {
		return ctx, fmt.Errorf("lookup project %s for worker context: %w", projectID, err)
	}
	orgID := strings.TrimSpace(project.OrganizationID)
	if orgID == "" {
		return ctx, fmt.Errorf("%w: project %s has no organization", domain.ErrInvalidInput, projectID)
	}
	return WithRequestAuthContext(ctx, RequestAuthContext{
		OrganizationID: orgID,
		Role:           RoleWorker,
	}), nil
}

// workerJobAuthContextForTimeline 通过 timeline -> episode -> project 链路
// 解析 export 所属组织，并派生 worker auth context。失败语义与
// workerJobAuthContextForProject 相同。
func (s *ProductionService) workerJobAuthContextForTimeline(ctx context.Context, timelineID string) (context.Context, error) {
	if s.projectSvc == nil {
		return ctx, nil
	}
	if strings.TrimSpace(timelineID) == "" {
		return ctx, fmt.Errorf("%w: worker export missing timeline id", domain.ErrInvalidInput)
	}
	timeline, err := s.production.GetTimelineByID(ctx, timelineID)
	if err != nil {
		return ctx, fmt.Errorf("lookup timeline %s for worker context: %w", timelineID, err)
	}
	if strings.TrimSpace(timeline.EpisodeID) == "" {
		return ctx, fmt.Errorf("%w: timeline %s has no episode", domain.ErrInvalidInput, timelineID)
	}
	episode, err := s.projectSvc.LookupEpisodeByID(ctx, timeline.EpisodeID)
	if err != nil {
		return ctx, fmt.Errorf("lookup episode %s for worker context: %w", timeline.EpisodeID, err)
	}
	return s.workerJobAuthContextForProject(ctx, episode.ProjectID)
}
