package jobs

type QueueName string
type JobKind string

const (
	QueueDefault QueueName = "default"

	DefaultExecutionLimit = 10
	DefaultPollInterval   = 5

	JobKindWorkflowSchedule   JobKind = "workflow.schedule"
	JobKindGenerationSubmit   JobKind = "generation.submit"
	JobKindGenerationPollTick JobKind = "generation.poll_tick"
	JobKindRetryGeneration    JobKind = "generation.retry"
	JobKindExportRender       JobKind = "export.render"
	JobKindEmailDistribution  JobKind = "email.distribution"
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
