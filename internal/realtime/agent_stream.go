package realtime

type AgentStreamEventType string

const (
	AgentStreamEventReasoning    AgentStreamEventType = "REASONING"
	AgentStreamEventContent      AgentStreamEventType = "CONTENT"
	AgentStreamEventToolCall     AgentStreamEventType = "TOOL_CALL"
	AgentStreamEventToolFinished AgentStreamEventType = "TOOL_FINISHED"
	AgentStreamEventDone         AgentStreamEventType = "DONE"
	AgentStreamEventError        AgentStreamEventType = "ERROR"
	AgentStreamEventCancelled    AgentStreamEventType = "CANCELLED"
)

type AgentToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type AgentStreamEvent struct {
	Type             AgentStreamEventType `json:"type"`
	RunID            string               `json:"run_id,omitempty"`
	Sequence         int64                `json:"sequence,omitempty"`
	Replay           bool                 `json:"replay,omitempty"`
	OccurredAt       string               `json:"occurred_at,omitempty"`
	Role             string               `json:"role,omitempty"`
	Content          string               `json:"content,omitempty"`
	Output           string               `json:"output,omitempty"`
	Highlights       []string             `json:"highlights,omitempty"`
	TokenCount       int                  `json:"token_count,omitempty"`
	DurationMS       int64                `json:"duration_ms,omitempty"`
	Error            string               `json:"error,omitempty"`
	ToolCalls        []AgentToolCall      `json:"tool_calls,omitempty"`
	ToolCallID       string               `json:"tool_call_id,omitempty"`
	ToolName         string               `json:"tool_name,omitempty"`
	ToolStatus       string               `json:"tool_status,omitempty"`
	ToolResult       string               `json:"tool_result,omitempty"`
	AgentName        string               `json:"agent_name,omitempty"`
	ParentToolCallID string               `json:"parent_tool_call_id,omitempty"`
}
