package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/workflow"
)

func (s *ProductionService) completeGeneratedStoryAnalysis(
	ctx context.Context,
	generationJob domain.GenerationJob,
) (domain.StoryAnalysis, error) {
	if err := generationJob.Status.ValidateTransition(domain.GenerationJobStatusSucceeded); err != nil {
		return domain.StoryAnalysis{}, err
	}
	source, err := s.latestStorySourceOrDefault(ctx, generationJob)
	if err != nil {
		return domain.StoryAnalysis{}, err
	}

	var analysisParams repo.CreateStoryAnalysisParams
	if s.agentSvc != nil && s.agentSvc.IsAvailable(ctx) {
		analysisParams, err = s.runLLMStoryAnalysis(ctx, generationJob, source)
	} else {
		analysisParams, err = s.runLocalStoryAnalysis(ctx, generationJob, source)
	}
	if err != nil {
		return domain.StoryAnalysis{}, err
	}

	completion, err := s.production.CompleteStoryAnalysisJob(ctx, repo.CompleteStoryAnalysisJobParams{
		Job: repo.AdvanceGenerationJobStatusParams{
			ID:           generationJob.ID,
			From:         generationJob.Status,
			To:           domain.GenerationJobStatusSucceeded,
			EventMessage: "story analysis completed",
		},
		Analysis: analysisParams,
	})
	if err != nil {
		return domain.StoryAnalysis{}, err
	}
	return completion.StoryAnalysis, nil
}

func (s *ProductionService) runLLMStoryAnalysis(
	ctx context.Context,
	job domain.GenerationJob,
	source domain.StorySource,
) (repo.CreateStoryAnalysisParams, error) {
	executor := s.agentSvc.MakeNodeExecutor(source.ContentText)
	return s.runStoryAnalysisWorkflow(ctx, job, source, executor, buildLLMSummary)
}

func (s *ProductionService) runLocalStoryAnalysis(
	ctx context.Context,
	job domain.GenerationJob,
	source domain.StorySource,
) (repo.CreateStoryAnalysisParams, error) {
	analysis := analyzeStorySource(source)
	return s.runStoryAnalysisWorkflow(
		ctx,
		job,
		source,
		makeDeterministicStoryAnalysisExecutor(analysis),
		func(_ domain.StorySource, _ *workflow.Blackboard) string { return analysis.summary },
	)
}

func (s *ProductionService) runStoryAnalysisWorkflow(
	ctx context.Context,
	job domain.GenerationJob,
	source domain.StorySource,
	executor workflow.NodeExecutor,
	summaryBuilder func(domain.StorySource, *workflow.Blackboard) string,
) (repo.CreateStoryAnalysisParams, error) {
	bb := workflow.NewBlackboard()
	engine := workflow.NewEngine(workflow.Phase1Graph, bb, executor)
	if strings.TrimSpace(job.WorkflowRunID) != "" {
		checkpointStore := newStoryAnalysisCheckpointStore(s.production)
		engine.EnableCheckpointing(job.WorkflowRunID, checkpointStore)
		checkpoint, err := checkpointStore.Load(ctx, job.WorkflowRunID)
		if err != nil {
			return repo.CreateStoryAnalysisParams{}, fmt.Errorf("load workflow checkpoint: %w", err)
		}
		if checkpoint != nil {
			if err := engine.Resume(checkpoint); err != nil {
				return repo.CreateStoryAnalysisParams{}, fmt.Errorf("resume workflow checkpoint: %w", err)
			}
		}
	}

	if err := engine.Execute(ctx); err != nil {
		return repo.CreateStoryAnalysisParams{}, fmt.Errorf("DAG execution: %w", err)
	}

	id, err := domain.NewID()
	if err != nil {
		return repo.CreateStoryAnalysisParams{}, err
	}

	themes, outline := parseLLMAnalysisResults(bb)
	characters, scenes, props := parseLLMMapResults(bb)

	return repo.CreateStoryAnalysisParams{
		ID:              id,
		ProjectID:       job.ProjectID,
		EpisodeID:       job.EpisodeID,
		StorySourceID:   source.ID,
		WorkflowRunID:   job.WorkflowRunID,
		GenerationJobID: job.ID,
		Status:          domain.StoryAnalysisStatusGenerated,
		Summary:         summaryBuilder(source, bb),
		Themes:          themes,
		CharacterSeeds:  characters,
		SceneSeeds:      scenes,
		PropSeeds:       props,
		Outline:         outline,
		AgentOutputs:    storyAnalysisAgentOutputsFromRuns(engine.Runs()),
	}, nil
}

func parseLLMAnalysisResults(bb *workflow.Blackboard) ([]string, []domain.StoryBeat) {
	themes := []string{"成长", "抉择"}
	if r, ok := bb.Read("story_analyst"); ok {
		if ar, ok := r.(*AgentResult); ok {
			themes = ar.Highlights
		}
	}

	outline := []domain.StoryBeat{
		{Code: "B01", Title: "开端", Summary: "故事开端"},
		{Code: "B02", Title: "发展", Summary: "故事发展"},
		{Code: "B03", Title: "转折", Summary: "故事转折"},
		{Code: "B04", Title: "高潮", Summary: "故事高潮"},
	}
	if r, ok := bb.Read("outline_planner"); ok {
		if ar, ok := r.(*AgentResult); ok {
			if parsed := parseBeatsFromJSON(ar.Output); len(parsed) > 0 {
				outline = parsed
			}
		}
	}
	return themes, outline
}

func parseLLMMapResults(bb *workflow.Blackboard) ([]string, []string, []string) {
	extract := func(role string) []string {
		r, ok := bb.Read(role)
		if !ok {
			return nil
		}
		ar, ok := r.(*AgentResult)
		if !ok {
			return nil
		}
		return ar.Highlights
	}
	return extract("character_analyst"), extract("scene_analyst"), extract("prop_analyst")
}

func parseBeatsFromJSON(content string) []domain.StoryBeat {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var data struct {
		Beats []domain.StoryBeat `json:"beats"`
	}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil
	}
	return data.Beats
}

func buildLLMSummary(source domain.StorySource, bb *workflow.Blackboard) string {
	prefix := "LLM 多 Agent 分析"
	if source.Title != "" {
		prefix = fmt.Sprintf("《%s》LLM 多 Agent 分析", source.Title)
	}
	if r, ok := bb.Read("story_analyst"); ok {
		if ar, ok := r.(*AgentResult); ok && len(ar.Highlights) > 0 {
			return prefix + "：" + strings.Join(ar.Highlights, "、")
		}
	}
	return prefix
}

func (s *ProductionService) ListStoryAnalyses(
	ctx context.Context,
	episodeID string,
) ([]domain.StoryAnalysis, error) {
	if strings.TrimSpace(episodeID) == "" {
		return nil, fmt.Errorf("%w: episode id is required", domain.ErrInvalidInput)
	}
	if err := s.authorizeEpisode(ctx, episodeID); err != nil {
		return nil, err
	}
	return s.production.ListStoryAnalyses(ctx, episodeID)
}

func (s *ProductionService) GetStoryAnalysis(ctx context.Context, id string) (domain.StoryAnalysis, error) {
	if strings.TrimSpace(id) == "" {
		return domain.StoryAnalysis{}, fmt.Errorf("%w: story analysis id is required", domain.ErrInvalidInput)
	}
	analysis, err := s.production.GetStoryAnalysis(ctx, id)
	if err != nil {
		return domain.StoryAnalysis{}, err
	}
	if err := s.authorizeScopedResource(ctx, analysis.ProjectID, analysis.EpisodeID); err != nil {
		return domain.StoryAnalysis{}, err
	}
	return analysis, nil
}

func (s *ProductionService) latestStoryAnalysis(ctx context.Context, episodeID string) (domain.StoryAnalysis, error) {
	analyses, err := s.ListStoryAnalyses(ctx, episodeID)
	if err != nil {
		return domain.StoryAnalysis{}, err
	}
	if len(analyses) == 0 {
		return domain.StoryAnalysis{}, domain.ErrNotFound
	}
	return analyses[0], nil
}

func (s *ProductionService) latestStorySourceOrDefault(
	ctx context.Context,
	generationJob domain.GenerationJob,
) (domain.StorySource, error) {
	source, err := s.production.LatestStorySource(ctx, generationJob.EpisodeID)
	if errors.Is(err, domain.ErrNotFound) {
		return defaultStorySource(generationJob), nil
	}
	return source, err
}

func defaultStorySource(generationJob domain.GenerationJob) domain.StorySource {
	return domain.StorySource{
		ProjectID:   generationJob.ProjectID,
		EpisodeID:   generationJob.EpisodeID,
		Title:       "默认故事样例",
		ContentText: "少年在云端宗门发现天门试炼的秘密。对立势力逼近，他必须在守护同伴和追寻身世之间做出选择。最终主角凭借关键线索完成试炼。",
		Language:    "zh-CN",
	}
}

func storyAnalysisAgentOutputsFromRuns(runs map[string]*workflow.NodeRun) []domain.StoryAgentOutput {
	agentOutputs := make([]domain.StoryAgentOutput, 0, len(runs))
	for _, node := range workflow.Phase1Graph.Nodes {
		run := runs[node.ID]
		output := domain.StoryAgentOutput{Role: node.ID, Status: string(run.Status)}
		if r, ok := run.Output.(*AgentResult); ok {
			output.Output = r.Output
			output.Highlights = append([]string(nil), r.Highlights...)
		}
		if run.Error != nil {
			output.Output = run.Error.Error()
		}
		agentOutputs = append(agentOutputs, output)
	}
	return agentOutputs
}
