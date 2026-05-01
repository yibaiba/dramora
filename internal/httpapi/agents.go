package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type streamAgentRunRequest struct {
	Role       string            `json:"role"`
	SourceText string            `json:"source_text"`
	Context    map[string]string `json:"context,omitempty"`
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

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")
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

	ctx := r.Context()
	result, runErr := a.agentService.RunSingleAgentStream(ctx, req.Role, req.SourceText, req.Context, func(delta string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return writeFrame("delta", map[string]string{"content": delta})
	})

	if runErr != nil {
		if errors.Is(runErr, ctx.Err()) && ctx.Err() != nil {
			return
		}
		_ = writeFrame("error", map[string]string{"message": runErr.Error()})
		return
	}

	_ = writeFrame("done", streamAgentDoneFrame{
		Role:       result.Role,
		Output:     result.Output,
		Highlights: result.Highlights,
		TokenCount: result.TokenCount,
		DurationMS: result.DurationMS,
	})
}
