package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/jobs"
	"github.com/yibaiba/dramora/internal/provider"
	"github.com/yibaiba/dramora/internal/repo"
)

const (
	sd2DefaultNegativePrompt = "no flicker, no jitter, no face distortion, no extra fingers, no costume drift, no unreadable text"
	maxSD2References         = 9
)

func (s *ProductionService) GenerateShotPromptPack(
	ctx context.Context,
	shotID string,
) (domain.ShotPromptPack, error) {
	if strings.TrimSpace(shotID) == "" {
		return domain.ShotPromptPack{}, fmt.Errorf("%w: shot id is required", domain.ErrInvalidInput)
	}
	shot, err := s.production.GetStoryboardShot(ctx, shotID)
	if err != nil {
		return domain.ShotPromptPack{}, err
	}
	if err := s.authorizeScopedResource(ctx, shot.ProjectID, shot.EpisodeID); err != nil {
		return domain.ShotPromptPack{}, err
	}
	assets, err := s.production.ListAssetsByEpisode(ctx, shot.EpisodeID)
	if err != nil {
		return domain.ShotPromptPack{}, err
	}
	packParams, err := buildShotPromptPackParams(shot, readyReferenceAssets(assets))
	if err != nil {
		return domain.ShotPromptPack{}, err
	}
	return s.production.SaveShotPromptPack(ctx, packParams)
}

func (s *ProductionService) GetShotPromptPack(
	ctx context.Context,
	shotID string,
) (domain.ShotPromptPack, error) {
	if strings.TrimSpace(shotID) == "" {
		return domain.ShotPromptPack{}, fmt.Errorf("%w: shot id is required", domain.ErrInvalidInput)
	}
	pack, err := s.production.GetShotPromptPack(ctx, shotID)
	if err != nil {
		return domain.ShotPromptPack{}, err
	}
	if err := s.authorizeScopedResource(ctx, pack.ProjectID, pack.EpisodeID); err != nil {
		return domain.ShotPromptPack{}, err
	}
	return pack, nil
}

type SaveShotPromptPackInput struct {
	DirectPrompt string
}

func (s *ProductionService) SaveShotPromptPack(
	ctx context.Context,
	shotID string,
	input SaveShotPromptPackInput,
) (domain.ShotPromptPack, error) {
	if strings.TrimSpace(shotID) == "" {
		return domain.ShotPromptPack{}, fmt.Errorf("%w: shot id is required", domain.ErrInvalidInput)
	}
	directPrompt := strings.TrimSpace(input.DirectPrompt)
	if directPrompt == "" {
		return domain.ShotPromptPack{}, fmt.Errorf("%w: direct prompt is required", domain.ErrInvalidInput)
	}
	pack, err := s.production.GetShotPromptPack(ctx, shotID)
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return domain.ShotPromptPack{}, err
		}
		shot, shotErr := s.production.GetStoryboardShot(ctx, shotID)
		if shotErr != nil {
			return domain.ShotPromptPack{}, shotErr
		}
		if err := s.authorizeScopedResource(ctx, shot.ProjectID, shot.EpisodeID); err != nil {
			return domain.ShotPromptPack{}, err
		}
		packParams, buildErr := buildShotPromptPackParams(shot, nil)
		if buildErr != nil {
			return domain.ShotPromptPack{}, buildErr
		}
		packParams.DirectPrompt = directPrompt
		return s.production.SaveShotPromptPack(ctx, packParams)
	}
	if err := s.authorizeScopedResource(ctx, pack.ProjectID, pack.EpisodeID); err != nil {
		return domain.ShotPromptPack{}, err
	}
	return s.production.SaveShotPromptPack(ctx, repo.SaveShotPromptPackParams{
		ID:                pack.ID,
		ProjectID:         pack.ProjectID,
		EpisodeID:         pack.EpisodeID,
		ShotID:            pack.ShotID,
		Provider:          pack.Provider,
		Model:             pack.Model,
		Preset:            pack.Preset,
		TaskType:          pack.TaskType,
		DirectPrompt:      directPrompt,
		NegativePrompt:    pack.NegativePrompt,
		TimeSlices:        pack.TimeSlices,
		ReferenceBindings: pack.ReferenceBindings,
		Params:            pack.Params,
	})
}

func (s *ProductionService) StartShotVideoGeneration(
	ctx context.Context,
	shotID string,
) (domain.GenerationJob, error) {
	if strings.TrimSpace(shotID) == "" {
		return domain.GenerationJob{}, fmt.Errorf("%w: shot id is required", domain.ErrInvalidInput)
	}
	pack, err := s.production.GetShotPromptPack(ctx, shotID)
	if err != nil {
		return domain.GenerationJob{}, err
	}
	if err := s.authorizeScopedResource(ctx, pack.ProjectID, pack.EpisodeID); err != nil {
		return domain.GenerationJob{}, err
	}
	jobID, err := domain.NewID()
	if err != nil {
		return domain.GenerationJob{}, err
	}
	job, err := s.production.CreateGenerationJob(ctx, repo.CreateGenerationJobParams{
		ID: jobID, ProjectID: pack.ProjectID, EpisodeID: pack.EpisodeID,
		RequestKey:   "shot-video:" + pack.ShotID + ":" + pack.Preset,
		Provider:     pack.Provider,
		Model:        pack.Model,
		TaskType:     pack.TaskType,
		Status:       domain.GenerationJobStatusQueued,
		Prompt:       pack.DirectPrompt,
		Params:       generationJobParamsFromPromptPack(pack),
		EventMessage: "shot video generation queued",
	})
	if err != nil {
		return domain.GenerationJob{}, err
	}
	if job.ID != jobID {
		return job, nil
	}
	if err := s.jobClient.Enqueue(ctx, jobs.Job{
		ID:   job.ID,
		Kind: jobs.JobKindGenerationSubmit,
		Payload: map[string]any{
			"generation_job_id": job.ID,
			"prompt_pack_id":    pack.ID,
			"shot_id":           pack.ShotID,
		},
	}); err != nil {
		return domain.GenerationJob{}, err
	}
	return job, nil
}

func generationJobParamsFromPromptPack(pack domain.ShotPromptPack) map[string]any {
	params := clonePromptParams(pack.Params)
	params["prompt_pack_id"] = pack.ID
	params["shot_id"] = pack.ShotID
	params["preset"] = pack.Preset
	params["reference_bindings"] = pack.ReferenceBindings
	params["time_slices"] = pack.TimeSlices
	return params
}

func buildShotPromptPackParams(
	shot domain.StoryboardShot,
	refs []domain.Asset,
) (repo.SaveShotPromptPackParams, error) {
	id, err := domain.NewID()
	if err != nil {
		return repo.SaveShotPromptPackParams{}, err
	}
	profile := provider.SeedanceFastProfile()
	bindings := promptReferenceBindings(refs)
	durationSec := promptDurationSeconds(shot.DurationMS)
	taskType := provider.TaskTypeTextToVideo
	if len(bindings) > 0 {
		taskType = provider.TaskTypeImageToVideo
	}
	return repo.SaveShotPromptPackParams{
		ID: id, ProjectID: shot.ProjectID, EpisodeID: shot.EpisodeID, ShotID: shot.ID,
		Provider: profile.Provider, Model: profile.Model, Preset: profile.Preset,
		TaskType: string(taskType), DirectPrompt: directSD2Prompt(shot, bindings),
		NegativePrompt:    sd2DefaultNegativePrompt,
		TimeSlices:        promptTimeSlices(shot),
		ReferenceBindings: bindings,
		Params:            promptPackParams(profile, taskType, durationSec, bindings),
	}, nil
}

func readyReferenceAssets(assets []domain.Asset) []domain.Asset {
	refs := make([]domain.Asset, 0, len(assets))
	for _, asset := range assets {
		if asset.Status == domain.AssetStatusReady {
			refs = append(refs, asset)
		}
	}
	sort.Slice(refs, func(i int, j int) bool {
		if refs[i].Kind == refs[j].Kind {
			return refs[i].URI < refs[j].URI
		}
		return refs[i].Kind < refs[j].Kind
	})
	if len(refs) > maxSD2References {
		return refs[:maxSD2References]
	}
	return refs
}

func promptReferenceBindings(refs []domain.Asset) []domain.PromptReferenceBinding {
	bindings := make([]domain.PromptReferenceBinding, 0, len(refs))
	for index, asset := range refs {
		role := "reference_image"
		if index == 0 {
			role = "first_frame"
		}
		bindings = append(bindings, domain.PromptReferenceBinding{
			Token: fmt.Sprintf("@image%d", index+1), Role: role,
			AssetID: asset.ID, Kind: asset.Kind, URI: asset.URI,
		})
	}
	return bindings
}

func promptDurationSeconds(durationMS int) int {
	if durationMS <= 0 {
		return provider.SeedanceFastProfile().DefaultDurationSec
	}
	seconds := (durationMS + 999) / 1000
	if seconds > 15 {
		return 15
	}
	return seconds
}

func directSD2Prompt(shot domain.StoryboardShot, refs []domain.PromptReferenceBinding) string {
	parts := []string{
		fmt.Sprintf("%s %s: %s", shot.Code, shot.Title, shot.Description),
		shot.Prompt,
		"AI manju cinematic panel, expressive composition, consistent character design, coherent scene lighting.",
		"Camera: slow push-in with stable framing; Style: high quality animated comic video.",
		"Stability: smooth motion, stable face, consistent outfit, normal anatomy, no flicker.",
	}
	if len(refs) > 0 {
		parts = append(parts, "Use locked references "+referenceTokenList(refs)+"; keep @image2 and later refs as continuity anchors.")
	}
	return strings.Join(nonEmptyStrings(parts), " ")
}

func promptTimeSlices(shot domain.StoryboardShot) []domain.PromptTimeSlice {
	duration := shot.DurationMS
	if duration <= 0 {
		duration = provider.SeedanceFastProfile().DefaultDurationSec * 1000
	}
	firstEnd := duration / 3
	secondEnd := (duration * 2) / 3
	return []domain.PromptTimeSlice{
		{StartMS: 0, EndMS: firstEnd, Prompt: "Establish the subject and scene mood.", CameraWork: "slow push-in", ShotSize: "wide", VisualFocus: shot.Title},
		{StartMS: firstEnd, EndMS: secondEnd, Prompt: "Advance the key action with clear motion.", CameraWork: "gentle tracking", ShotSize: "medium", VisualFocus: shot.Description},
		{StartMS: secondEnd, EndMS: duration, Prompt: "Hold a clean final pose for continuity.", CameraWork: "stable hold", ShotSize: "close", VisualFocus: "last frame continuity"},
	}
}

func promptPackParams(
	profile provider.ModelProfile,
	taskType provider.TaskType,
	durationSec int,
	refs []domain.PromptReferenceBinding,
) map[string]any {
	return map[string]any{
		"ratio":             profile.DefaultRatio,
		"resolution":        profile.DefaultResolution,
		"duration":          durationSec,
		"service_tier":      profile.ServiceTier,
		"return_last_frame": true,
		"watermark":         false,
		"provider_mode":     "fake_default_real_when_ark_key_present",
		"task_type":         string(taskType),
		"reference_count":   len(refs),
	}
}

func referenceTokenList(refs []domain.PromptReferenceBinding) string {
	tokens := make([]string, 0, len(refs))
	for _, ref := range refs {
		tokens = append(tokens, ref.Token)
	}
	return strings.Join(tokens, ", ")
}

func nonEmptyStrings(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func clonePromptParams(values map[string]any) map[string]any {
	if values == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(values)+4)
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

// PromptPackRecoveryJob 是 PromptPackRecovery 中每个关联生成任务的快照。
type PromptPackRecoveryJob struct {
	Job     domain.GenerationJob
	Summary GenerationJobRecoverySummary
}

// PromptPackRecoverySummary 汇总 prompt pack 视角下所有关联生成任务的可恢复状态。
type PromptPackRecoverySummary struct {
	JobsTotal           int
	TerminalCount       int
	RecoverableCount    int
	InFlightCount       int
	LastEventAt         time.Time
	HasRecoverable      bool
	NextHint            string
	LatestStatus        domain.GenerationJobStatus
	LatestStatusJobID   string
	LatestStatusJobTime time.Time
}

// PromptPackRecovery 描述 shot prompt pack 与其触发的生成任务集合的恢复视图。
// 不引入新的 events 表：每个 job 复用 GenerationJobEvent，pack-level summary 由各 job recovery summary 汇总而成。
type PromptPackRecovery struct {
	Pack    domain.ShotPromptPack
	Jobs    []PromptPackRecoveryJob
	Summary PromptPackRecoverySummary
}

func (s *ProductionService) GetShotPromptPackRecovery(
	ctx context.Context,
	shotID string,
) (PromptPackRecovery, error) {
	pack, err := s.GetShotPromptPack(ctx, shotID)
	if err != nil {
		return PromptPackRecovery{}, err
	}
	allJobs, err := s.production.ListGenerationJobs(ctx)
	if err != nil {
		return PromptPackRecovery{}, err
	}
	matched := make([]PromptPackRecoveryJob, 0)
	for _, job := range allJobs {
		if !generationJobMatchesPromptPack(job, pack) {
			continue
		}
		events, evErr := s.production.ListGenerationJobEvents(ctx, job.ID, 0)
		if evErr != nil {
			return PromptPackRecovery{}, evErr
		}
		matched = append(matched, PromptPackRecoveryJob{
			Job:     job,
			Summary: buildGenerationJobRecoverySummary(job, events),
		})
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Job.CreatedAt.After(matched[j].Job.CreatedAt)
	})
	return PromptPackRecovery{
		Pack:    pack,
		Jobs:    matched,
		Summary: buildPromptPackRecoverySummary(matched),
	}, nil
}

func generationJobMatchesPromptPack(job domain.GenerationJob, pack domain.ShotPromptPack) bool {
	if job.Params == nil {
		return false
	}
	if v, ok := job.Params["prompt_pack_id"]; ok {
		if s, _ := v.(string); s != "" && s == pack.ID {
			return true
		}
	}
	if v, ok := job.Params["shot_id"]; ok {
		if s, _ := v.(string); s != "" && s == pack.ShotID {
			return true
		}
	}
	return false
}

func buildPromptPackRecoverySummary(jobs []PromptPackRecoveryJob) PromptPackRecoverySummary {
	summary := PromptPackRecoverySummary{JobsTotal: len(jobs)}
	for _, item := range jobs {
		if item.Summary.IsTerminal {
			summary.TerminalCount++
		}
		if item.Summary.IsRecoverable {
			summary.RecoverableCount++
			summary.HasRecoverable = true
		}
		if !item.Summary.IsTerminal && !item.Summary.IsRecoverable {
			summary.InFlightCount++
		}
		if item.Summary.LastEventAt.After(summary.LastEventAt) {
			summary.LastEventAt = item.Summary.LastEventAt
		}
		if item.Job.UpdatedAt.After(summary.LatestStatusJobTime) {
			summary.LatestStatus = item.Job.Status
			summary.LatestStatusJobID = item.Job.ID
			summary.LatestStatusJobTime = item.Job.UpdatedAt
		}
	}
	switch {
	case summary.JobsTotal == 0:
		summary.NextHint = "未触发生成任务"
	case summary.HasRecoverable:
		summary.NextHint = "存在可恢复任务，建议人工介入或等待 worker 重试"
	case summary.InFlightCount > 0:
		summary.NextHint = "仍有任务在进行，等待完成"
	case summary.TerminalCount == summary.JobsTotal:
		summary.NextHint = "全部终态，可考虑重新触发"
	}
	return summary
}
