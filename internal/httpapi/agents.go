package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/yibaiba/dramora/internal/realtime"
	"github.com/yibaiba/dramora/internal/service"
)

type streamAgentRunRequest struct {
	Role       string            `json:"role"`
	SourceText string            `json:"source_text"`
	Context    map[string]string `json:"context,omitempty"`
	EpisodeID  string            `json:"episode_id,omitempty"`
}

type streamAgentDoneFrame struct {
	Role       string   `json:"role"`
	Output     string   `json:"output"`
	Highlights []string `json:"highlights"`
	TokenCount int      `json:"token_count"`
	DurationMS int64    `json:"duration_ms"`
}

func (a *api) streamAgentRun(w http.ResponseWriter, r *http.Request) {
	if a.agentService == nil {
		writeError(w, http.StatusServiceUnavailable, "agent_unavailable", "agent service not configured")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream_not_supported", "streaming is not supported")
		return
	}

	var req streamAgentRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if req.Role == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "role required")
		return
	}
	if req.SourceText == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "source_text required")
		return
	}
	if !a.agentService.SupportsRole(req.Role) {
		writeError(w, http.StatusBadRequest, "invalid_request", "unsupported agent role")
		return
	}

	// 组织隔离验证：
	// - 如果指定了 episodeID，验证用户可以访问该 episode（自动检查组织隔离）
	// - 如果没有指定，则此端点仅允许 admin 使用（诊断/演示目的）
	ctx := r.Context()
	if req.EpisodeID != "" {
		if _, err := a.projectService.GetEpisode(ctx, req.EpisodeID); err != nil {
			writeServiceError(w, err)
			return
		}
	} else {
		// 无 episode context 时需要 admin/owner 权限
		auth, ok := service.RequestAuthFromContext(ctx)
		if !ok || (auth.Role != "admin" && auth.Role != "owner") {
			writeError(w, http.StatusForbidden, "permission_denied", "admin/owner role required for unrestricted agent stream")
			return
		}
	}

	run := a.agentRunManager.StartRun(req.Role, req.EpisodeID)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Agent-Run-ID", run.RunID)
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	writeFrame := func(event string, payload any) error {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	result, runErr := a.agentService.RunSingleAgentStream(ctx, req.Role, req.SourceText, req.Context, func(delta string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := writeFrame("delta", map[string]string{"content": delta}); err != nil {
			return err
		}
		event, _, ok := a.agentRunManager.AppendEvent(run.RunID, realtime.AgentStreamEvent{
			Type:    realtime.AgentStreamEventContent,
			Role:    req.Role,
			Content: delta,
		})
		if !ok {
			return errors.New("agent run not found")
		}
		return writeFrame("agent_event", event)
	})

	if runErr != nil {
		if errors.Is(runErr, ctx.Err()) && ctx.Err() != nil {
			return
		}
		_ = writeFrame("error", map[string]string{"message": runErr.Error()})
		event, _, ok := a.agentRunManager.AppendEvent(run.RunID, realtime.AgentStreamEvent{
			Type:  realtime.AgentStreamEventError,
			Role:  req.Role,
			Error: runErr.Error(),
		})
		if ok {
			_ = writeFrame("agent_event", event)
		}
		return
	}

	_ = writeFrame("done", streamAgentDoneFrame{
		Role:       result.Role,
		Output:     result.Output,
		Highlights: result.Highlights,
		TokenCount: result.TokenCount,
		DurationMS: result.DurationMS,
	})
	event, _, ok := a.agentRunManager.AppendEvent(run.RunID, realtime.AgentStreamEvent{
		Type:       realtime.AgentStreamEventDone,
		Role:       result.Role,
		Output:     result.Output,
		Highlights: result.Highlights,
		TokenCount: result.TokenCount,
		DurationMS: result.DurationMS,
	})
	if ok {
		_ = writeFrame("agent_event", event)
	}
}

func (a *api) getAgentRun(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runId")
	if runID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "run id required")
		return
	}
	snapshot, ok := a.agentRunManager.GetSnapshot(runID)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "agent run not found")
		return
	}
	if !a.authorizeAgentRun(w, r, snapshot) {
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (a *api) reconnectAgentRunStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream_not_supported", "streaming is not supported")
		return
	}
	runID := chi.URLParam(r, "runId")
	if runID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "run id required")
		return
	}
	after := int64(0)
	if raw := r.URL.Query().Get("after"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 0 {
			writeError(w, http.StatusBadRequest, "invalid_request", "after must be a non-negative integer")
			return
		}
		after = parsed
	}
	replay, updates, cancel, snapshot, found := a.agentRunManager.SubscribeFrom(runID, after)
	if !found {
		writeError(w, http.StatusNotFound, "not_found", "agent run not found")
		return
	}
	defer cancel()
	if !a.authorizeAgentRun(w, r, snapshot) {
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Agent-Run-ID", runID)
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	writeFrame := func(event string, payload any) error {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	for _, event := range replay {
		if err := writeFrame("agent_event", event); err != nil {
			return
		}
	}
	if updates == nil || snapshot.Status != realtime.AgentRunStatusRunning {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-updates:
			if !ok {
				return
			}
			if err := writeFrame("agent_event", event); err != nil {
				return
			}
		}
	}
}

func (a *api) authorizeAgentRun(w http.ResponseWriter, r *http.Request, snapshot realtime.AgentRunSnapshot) bool {
	ctx := r.Context()
	if snapshot.EpisodeID != "" {
		if _, err := a.projectService.GetEpisode(ctx, snapshot.EpisodeID); err != nil {
			writeServiceError(w, err)
			return false
		}
		return true
	}
	auth, ok := service.RequestAuthFromContext(ctx)
	if !ok || (auth.Role != "admin" && auth.Role != "owner") {
		writeError(w, http.StatusForbidden, "permission_denied", "admin/owner role required for unrestricted agent stream")
		return false
	}
	return true
}
