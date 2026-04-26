package jobs

type QueueName string
type JobKind string

const (
	QueueDefault QueueName = "default"

	DefaultExecutionLimit = 10
	DefaultPollInterval   = 5

	JobKindWorkflowSchedule JobKind = "workflow.schedule"
	JobKindGenerationSubmit JobKind = "generation.submit"
	JobKindExportRender     JobKind = "export.render"
)

type Job struct {
	ID      string
	Kind    JobKind
	Payload map[string]any
}

type ExecutionSummary struct {
	Processed int
	Succeeded int
	Failed    int
}
