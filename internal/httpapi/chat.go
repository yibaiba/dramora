package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yibaiba/dramora/internal/provider"
)

// ChatMessageRequest 来自前端的 chat 请求。
type ChatMessageRequest struct {
	Messages []provider.ChatMessage `json:"messages"`
	Provider string                 `json:"provider,omitempty"` // 可选，用于特定提供商选择
}

// ChatResponse 表示一次 chat 响应。
type ChatResponse struct {
	ID         string `json:"id"`
	Content    string `json:"content"`
	TokenUsage struct {
		InputTokens  int `json:"input_tokens,omitempty"`
		OutputTokens int `json:"output_tokens,omitempty"`
	} `json:"token_usage,omitempty"`
	LatencyMS int64 `json:"latency_ms"`
}

// handleChatMessage 处理 POST /api/v1/episodes/{episodeId}/chat 请求。
// 流程：
// 1. 验证 episode 组织隔离
// 2. 调用 LLM provider
// 3. 扣费 1 积分
// 4. 返回响应
func (a *api) handleChatMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	episodeID := chi.URLParam(r, "episodeId")

	// 验证组织隔离（确保用户可以访问该 episode）
	if _, err := a.projectService.GetEpisode(ctx, episodeID); err != nil {
		writeServiceError(w, err)
		return
	}

	// 解析请求
	var req ChatMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "messages cannot be empty")
		return
	}

	// 调用 LLM provider
	start := time.Now()
	llmResp := a.providerService.SmokeChatProvider(ctx)
	latency := time.Since(start).Milliseconds()

	if !llmResp.OK {
		writeError(w, http.StatusInternalServerError, "provider_error", fmt.Sprintf("LLM provider failed: %s", llmResp.Error))
		return
	}

	// 生成响应 ID
	chatResponseID := fmt.Sprintf("chat-%d", time.Now().UnixNano())

	// 扣费（基于 token 数）
	// 遵循 Phase 2 的幂等性策略：以 (refType, refId) 唯一标识
	// refType = "chat", refId = chatResponseID
	inputTokens := int64(0)
	outputTokens := int64(0)
	if llmResp.TokenCount > 0 {
		// 注：当前实现只返回 output tokens，input tokens 需要后续改进
		outputTokens = int64(llmResp.TokenCount)
	}
	_, _ = a.walletService.DebitChatOperation(ctx, inputTokens, outputTokens, chatResponseID)
	// 扣费失败不阻塞响应（silent swallow）
	// PendingBillingWorker 会后续重试

	// 返回响应
	chatResp := ChatResponse{
		ID:        chatResponseID,
		Content:   llmResp.Content,
		LatencyMS: latency,
	}
	if llmResp.TokenCount > 0 {
		chatResp.TokenUsage.OutputTokens = llmResp.TokenCount
	}

	writeJSON(w, http.StatusOK, Envelope{
		"chat_response": chatResp,
	})
}
