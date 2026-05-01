package provider

import (
	"context"
	"time"
)

// CapabilityConfig 是供 image/video/audio 三类工厂使用的最小公共配置。
// 与 LLMConfig 一样，保持 primitive-only，避免依赖 domain 层。
type CapabilityConfig struct {
	ProviderType string // 见各 capability 工厂支持矩阵；空值取默认
	BaseURL      string
	APIKey       string
	Model        string
	Timeout      time.Duration
}

// ImageRequest 描述一次图像生成请求。规模刻意保持最小，
// 后续按需扩展（reference image / mask / size 等）。
type ImageRequest struct {
	Prompt    string
	Model     string
	Width     int
	Height    int
	NumImages int
}

// ImageResult 是图像生成的最小返回形状。
type ImageResult struct {
	URLs []string // 公开 URL 或对象存储引用
	Raw  string   // 原始响应（best-effort，便于排错）
}

// ImageProvider 抽象图像生成能力。
type ImageProvider interface {
	Name() string
	Generate(ctx context.Context, req ImageRequest) (*ImageResult, error)
}

// VideoSubmitRequest / VideoTask 与现有 SeedanceAdapter 的提交-轮询语义对齐。
// 视频生成是异步的：Submit 返回 task，Poll 检查 status / result。
type VideoSubmitRequest struct {
	Prompt      string
	Model       string
	Ratio       string
	Resolution  string
	DurationSec int
	Seed        int
}

type VideoTask struct {
	ID        string
	Status    string // queued | running | succeeded | failed | ...
	Mode      string
	ResultURI string
}

// VideoProvider 抽象视频生成能力（异步任务模型）。
type VideoProvider interface {
	Name() string
	Submit(ctx context.Context, req VideoSubmitRequest) (VideoTask, error)
	Poll(ctx context.Context, taskID string) (VideoTask, error)
}

// AudioRequest 是 TTS 请求最小形态。
type AudioRequest struct {
	Text   string
	Model  string
	Voice  string
	Format string // mp3 | wav | ogg ...
}

// AudioResult 返回音频字节或 URL，二选一。
type AudioResult struct {
	URL   string
	Bytes []byte
	Raw   string
}

// AudioProvider 抽象 TTS / 语音合成能力。
type AudioProvider interface {
	Name() string
	Synthesize(ctx context.Context, req AudioRequest) (*AudioResult, error)
}
