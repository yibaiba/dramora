package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/provider"
	"github.com/yibaiba/dramora/internal/repo"
)

// resolveCapabilityConfig 把 provider_config（DB）转换成 provider.CapabilityConfig，
// 找不到（ErrNotFound）则返回 zero CapabilityConfig + 不报错，让工厂走默认 provider_type。
// 其它错误真实返回。
func (s *ProductionService) resolveCapabilityConfig(ctx context.Context, capability string) (provider.CapabilityConfig, error) {
	if s == nil || s.providerSvc == nil {
		return provider.CapabilityConfig{}, nil
	}
	cfg, err := s.providerSvc.GetProviderConfig(ctx, capability)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return provider.CapabilityConfig{}, nil
		}
		return provider.CapabilityConfig{}, err
	}
	return provider.CapabilityConfig{
		ProviderType: cfg.ResolvedProviderType(),
		BaseURL:      strings.TrimSpace(cfg.BaseURL),
		APIKey:       strings.TrimSpace(cfg.APIKey),
		Model:        strings.TrimSpace(cfg.Model),
	}, nil
}

func isImageGenerationJob(generationJob domain.GenerationJob) bool {
	return provider.TaskType(generationJob.TaskType) == provider.TaskTypeImage
}

func isAudioGenerationJob(generationJob domain.GenerationJob) bool {
	return provider.TaskType(generationJob.TaskType) == provider.TaskTypeTTS
}

// processImageGenerationJob 把 image capability 工厂接到 worker：
// 同步 HTTP 调用一次性返回 URL，状态机走 queued -> submitting -> submitted -> downloading -> postprocessing -> succeeded。
func (s *ProductionService) processImageGenerationJob(ctx context.Context, generationJob domain.GenerationJob) error {
	cfg, err := s.resolveCapabilityConfig(ctx, "image")
	if err != nil {
		_, _ = s.advanceGenerationJob(ctx, generationJob, domain.GenerationJobStatusFailed, "", "image worker config lookup failed")
		return err
	}
	imageProvider, err := provider.NewImageProvider(cfg)
	if err != nil {
		_, _ = s.advanceGenerationJob(ctx, generationJob, domain.GenerationJobStatusFailed, "", "image worker provider construction failed")
		return err
	}

	current := generationJob
	for _, step := range []struct {
		status  domain.GenerationJobStatus
		message string
	}{
		{domain.GenerationJobStatusSubmitting, "image worker submitting generation job"},
		{domain.GenerationJobStatusSubmitted, "image worker submitted generation job"},
	} {
		next, advErr := s.advanceGenerationJob(ctx, current, step.status, "", step.message)
		if advErr != nil {
			return advErr
		}
		current = next
	}

	model := strings.TrimSpace(stringParam(current.Params, "model"))
	if model == "" {
		model = cfg.Model
	}
	start := time.Now()
	result, err := imageProvider.Generate(ctx, provider.ImageRequest{
		Prompt:    current.Prompt,
		Model:     model,
		Width:     intParam(current.Params, "width"),
		Height:    intParam(current.Params, "height"),
		NumImages: intParam(current.Params, "n"),
	})
	s.recordCapabilityCall(start, "image", imageProvider.Name(), model, "generate", err)
	if err != nil {
		_, _ = s.advanceGenerationJob(ctx, current, domain.GenerationJobStatusFailed, "", "image worker generation failed")
		return err
	}
	if result == nil || len(result.URLs) == 0 {
		_, _ = s.advanceGenerationJob(ctx, current, domain.GenerationJobStatusFailed, "", "image worker missing result url")
		return fmt.Errorf("%w: image provider returned no urls", domain.ErrInvalidInput)
	}

	completed, err := s.completeMediaDownload(ctx, current, "image", "generated_image", result.URLs[0])
	if err != nil {
		return err
	}
	current = completed
	if current.Status == domain.GenerationJobStatusPostprocessing {
		_, err := s.advanceGenerationJob(ctx, current, domain.GenerationJobStatusSucceeded, "", "image worker completed generation job")
		if err == nil {
			// 成功完成后自动扣费
			s.debitOperationAfterSuccess(ctx, current.ID, domain.OperationTypeImageGeneration)
		}
		return err
	}
	return nil
}

// processAudioGenerationJob 把 audio capability 工厂接到 worker。
// openai TTS 默认返回原始音频字节；当前 MVP 把字节落到 placeholder data:// URI，
// 真正的对象存储集成留给后续 PR。当存在直接 URL 时优先使用 URL。
func (s *ProductionService) processAudioGenerationJob(ctx context.Context, generationJob domain.GenerationJob) error {
	cfg, err := s.resolveCapabilityConfig(ctx, "audio")
	if err != nil {
		_, _ = s.advanceGenerationJob(ctx, generationJob, domain.GenerationJobStatusFailed, "", "audio worker config lookup failed")
		return err
	}
	audioProvider, err := provider.NewAudioProvider(cfg)
	if err != nil {
		_, _ = s.advanceGenerationJob(ctx, generationJob, domain.GenerationJobStatusFailed, "", "audio worker provider construction failed")
		return err
	}

	current := generationJob
	for _, step := range []struct {
		status  domain.GenerationJobStatus
		message string
	}{
		{domain.GenerationJobStatusSubmitting, "audio worker submitting generation job"},
		{domain.GenerationJobStatusSubmitted, "audio worker submitted generation job"},
	} {
		next, advErr := s.advanceGenerationJob(ctx, current, step.status, "", step.message)
		if advErr != nil {
			return advErr
		}
		current = next
	}

	model := strings.TrimSpace(stringParam(current.Params, "model"))
	if model == "" {
		model = cfg.Model
	}
	start := time.Now()
	result, err := audioProvider.Synthesize(ctx, provider.AudioRequest{
		Text:   current.Prompt,
		Model:  model,
		Voice:  stringParam(current.Params, "voice"),
		Format: stringParam(current.Params, "format"),
	})
	s.recordCapabilityCall(start, "audio", audioProvider.Name(), model, "synthesize", err)
	if err != nil {
		_, _ = s.advanceGenerationJob(ctx, current, domain.GenerationJobStatusFailed, "", "audio worker synthesize failed")
		return err
	}
	resultURI, uriErr := s.audioResultURI(ctx, current, result)
	if uriErr != nil {
		_, _ = s.advanceGenerationJob(ctx, current, domain.GenerationJobStatusFailed, "", "audio worker failed to persist result bytes")
		return uriErr
	}
	resultURI = strings.TrimSpace(resultURI)
	if resultURI == "" {
		_, _ = s.advanceGenerationJob(ctx, current, domain.GenerationJobStatusFailed, "", "audio worker missing result uri")
		return fmt.Errorf("%w: audio provider returned no usable result", domain.ErrInvalidInput)
	}

	completed, err := s.completeMediaDownload(ctx, current, "audio", "generated_audio", resultURI)
	if err != nil {
		return err
	}
	current = completed
	if current.Status == domain.GenerationJobStatusPostprocessing {
		_, err := s.advanceGenerationJob(ctx, current, domain.GenerationJobStatusSucceeded, "", "audio worker completed generation job")
		return err
	}
	return nil
}

// audioResultURI 把 audio provider 的结果转换为可被资产层引用的 URI。
//
// 优先级：
//  1. provider 返回 URL —— 直接复用。
//  2. provider 返回 Bytes 且注入了 mediaStorage —— Put 到存储并使用返回 URI。
//  3. provider 返回 Bytes 但没有 mediaStorage —— 回落到 inline placeholder
//     （仅用于本地开发 / 老路径，不应在生产链路上长期保留）。
func (s *ProductionService) audioResultURI(
	ctx context.Context,
	job domain.GenerationJob,
	result *provider.AudioResult,
) (string, error) {
	if result == nil {
		return "", nil
	}
	if url := strings.TrimSpace(result.URL); url != "" {
		return url, nil
	}
	if len(result.Bytes) == 0 {
		return "", nil
	}
	if s == nil || s.mediaStorage == nil {
		return fmt.Sprintf("manmu://providers/audio/inline?bytes=%d", len(result.Bytes)), nil
	}
	ext := audioBytesExtension(stringParam(job.Params, "format"))
	key := fmt.Sprintf("audio/%s%s", job.ID, ext)
	obj, err := s.mediaStorage.Put(ctx, key, bytes.NewReader(result.Bytes), audioBytesContentType(ext))
	if err != nil {
		return "", fmt.Errorf("media storage put audio bytes: %w", err)
	}
	return obj.URI, nil
}

func audioBytesExtension(format string) string {
	f := strings.ToLower(strings.TrimSpace(format))
	switch f {
	case "", "mp3":
		return ".mp3"
	case "wav":
		return ".wav"
	case "flac":
		return ".flac"
	case "ogg":
		return ".ogg"
	case "opus":
		return ".opus"
	case "aac":
		return ".aac"
	case "m4a":
		return ".m4a"
	default:
		if strings.HasPrefix(f, ".") {
			return f
		}
		return "." + f
	}
}

func audioBytesContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".flac":
		return "audio/flac"
	case ".ogg":
		return "audio/ogg"
	case ".opus":
		return "audio/opus"
	case ".aac":
		return "audio/aac"
	case ".m4a":
		return "audio/mp4"
	default:
		return "application/octet-stream"
	}
}

// completeMediaDownload 是 image / audio worker 共用的资产持久化路径，
// 与 completeGenerationDownload(seedance video) 保持同样的状态变迁约束。
func (s *ProductionService) completeMediaDownload(
	ctx context.Context,
	generationJob domain.GenerationJob,
	assetKind string,
	assetPurpose string,
	resultURI string,
) (domain.GenerationJob, error) {
	current := generationJob
	if current.Status == domain.GenerationJobStatusSubmitted {
		next, err := s.advanceGenerationJob(ctx, current, domain.GenerationJobStatusDownloading, "", assetKind+" worker collecting generated output")
		if err != nil {
			return domain.GenerationJob{}, err
		}
		current = next
	}
	if current.Status != domain.GenerationJobStatusDownloading {
		return current, nil
	}
	if current.ResultAssetID != "" {
		return s.advanceGenerationJob(ctx, current, domain.GenerationJobStatusPostprocessing, "", assetKind+" worker postprocessing generated output")
	}
	if strings.TrimSpace(resultURI) == "" {
		return domain.GenerationJob{}, fmt.Errorf("%w: %s result uri is required", domain.ErrInvalidInput, assetKind)
	}
	assetID, err := domain.NewID()
	if err != nil {
		return domain.GenerationJob{}, err
	}
	job, _, err := s.production.CompleteGenerationJobWithResult(ctx, repo.CompleteGenerationJobWithResultParams{
		Job: repo.AdvanceGenerationJobStatusParams{
			ID: current.ID, From: current.Status, To: domain.GenerationJobStatusPostprocessing,
			EventMessage: assetKind + " worker downloaded result asset",
		},
		Asset: repo.CreateAssetParams{
			ID: assetID, ProjectID: current.ProjectID, EpisodeID: current.EpisodeID,
			Kind: assetKind, Purpose: assetPurpose, URI: resultURI,
			Status: domain.AssetStatusReady,
		},
	})
	return job, err
}

// recordCapabilityCall feeds a capability worker call event into the shared
// LLM telemetry buffer (when an AgentService is attached). Vendor and model
// are reported as observed at call time so admin telemetry can disambiguate
// e.g. mock vs openai vs seedance per capability.
func (s *ProductionService) recordCapabilityCall(start time.Time, capability, vendor, model, mode string, err error) {
	if s == nil || s.agentSvc == nil {
		return
	}
	ev := LLMTelemetryEvent{
		StartedAt:  start.UTC(),
		Capability: capability,
		Vendor:     strings.TrimSpace(vendor),
		Model:      strings.TrimSpace(model),
		Mode:       mode,
		DurationMS: time.Since(start).Milliseconds(),
		Success:    err == nil,
	}
	if err != nil {
		ev.ErrorMessage = err.Error()
	}
	s.agentSvc.RecordTelemetry(ev)
}
