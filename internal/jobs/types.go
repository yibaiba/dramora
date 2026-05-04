package jobs

type QueueName string
type JobKind string

const (
	QueueDefault QueueName = "default"

	DefaultExecutionLimit = 10
	DefaultPollInterval   = 5

	JobKindWorkflowSchedule     JobKind = "workflow.schedule"
	JobKindGenerationSubmit     JobKind = "generation.submit"
	JobKindGenerationPollTick   JobKind = "generation.poll_tick"
	JobKindRetryGeneration      JobKind = "generation.retry"
	JobKindExportRender         JobKind = "export.render"
	JobKindEmailDistribution    JobKind = "email.distribution"
	JobKindShortVideoGenerate   JobKind = "short_video.generate"
	JobKindShortVideoHeyGenPoll JobKind = "short_video.heygen_poll"
	JobKindShortVideoRetry      JobKind = "short_video.retry"
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
