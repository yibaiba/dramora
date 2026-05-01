package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/provider"
)

// SetProviderService 把 ProviderService 注入到 ProductionService，
// 让 video worker 可以从 provider_configs 取 video capability 配置，
// 不再硬依赖 env 构造的 SeedanceAdapter。
func (s *ProductionService) SetProviderService(providerSvc *ProviderService) {
	s.providerSvc = providerSvc
}

// resolveSeedance 在每次进入 video worker 路径时按以下优先级解析：
//  1. providerSvc != nil 且当前 ctx 能拿到 video capability 的 ProviderConfig：
//     - provider_type=seedance：用 config 重新构造 SeedanceAdapter（带 timeout）
//     - provider_type=mock：返回 deterministic mock seedanceProvider
//  2. 解析失败或没有配置：回退到启动时的 s.seedance（env 构造或显式注入）。
//
// 这样 admin 在 Studio 里把 video capability 切到 mock 时，
// worker 也会立刻走 mock 而不是真实 ARK 接口。
func (s *ProductionService) resolveSeedance(ctx context.Context) seedanceProvider {
	if s == nil {
		return nil
	}
	if s.providerSvc == nil {
		return s.seedance
	}
	cfg, err := s.providerSvc.GetProviderConfig(ctx, "video")
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			slog.WarnContext(ctx, "video provider config lookup failed; falling back to env seedance", "error", err)
		}
		return s.seedance
	}
	switch strings.ToLower(strings.TrimSpace(cfg.ResolvedProviderType())) {
	case "mock":
		return mockSeedanceProvider{}
	case "seedance", "":
		baseURL := strings.TrimSpace(cfg.BaseURL)
		if baseURL == "" {
			baseURL = provider.DefaultSeedanceArkBaseURL
		}
		client := &http.Client{Timeout: 60 * time.Second}
		return provider.NewSeedanceAdapter(strings.TrimSpace(cfg.APIKey), baseURL, client)
	default:
		slog.WarnContext(ctx, "video provider config has unsupported provider_type; falling back",
			"provider_type", cfg.ResolvedProviderType())
		return s.seedance
	}
}

// mockSeedanceProvider 是一个 deterministic 的 in-process 占位实现，
// 满足 seedanceProvider 接口。供 video capability provider_type=mock 时使用，
// 以及为单测提供离线版本。
type mockSeedanceProvider struct{}

func (mockSeedanceProvider) SubmitGeneration(_ context.Context, input provider.SeedanceRequestInput) (provider.SeedanceGenerationTask, error) {
	sum := sha1.Sum([]byte(input.Prompt + "|" + string(input.TaskType)))
	return provider.SeedanceGenerationTask{
		ID:     "mock-seedance-" + hex.EncodeToString(sum[:8]),
		Status: "queued",
		Mode:   "fake",
	}, nil
}

func (mockSeedanceProvider) PollGeneration(_ context.Context, taskID string) (provider.SeedanceGenerationTask, error) {
	if strings.TrimSpace(taskID) == "" {
		return provider.SeedanceGenerationTask{}, errors.New("seedance task id is required")
	}
	return provider.SeedanceGenerationTask{
		ID:        taskID,
		Status:    "succeeded",
		Mode:      "fake",
		ResultURI: "manmu://providers/mock-seedance/" + taskID + "/video.mp4",
	}, nil
}
