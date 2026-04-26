package realtime

type EventType string

const (
	EventWorkflowStatusChanged EventType = "workflow.status_changed"
	EventGenerationProgress    EventType = "generation.progress"
	EventGenerationCompleted   EventType = "generation.completed"
	EventGenerationFailed      EventType = "generation.failed"
	EventCostWarning           EventType = "cost.warning"
)

type Event struct {
	Type EventType      `json:"type"`
	Data map[string]any `json:"data"`
}
