package domain

import "fmt"

type ProjectStatus string
type EpisodeStatus string
type WorkflowRunStatus string
type WorkflowNodeRunStatus string
type AgentRunStatus string
type GenerationJobStatus string
type AssetStatus string
type ApprovalGateStatus string
type TimelineStatus string
type ExportStatus string

const (
	ProjectStatusDraft    ProjectStatus = "draft"
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusArchived ProjectStatus = "archived"

	EpisodeStatusDraft      EpisodeStatus = "draft"
	EpisodeStatusPlanning   EpisodeStatus = "planning"
	EpisodeStatusGenerating EpisodeStatus = "generating"
	EpisodeStatusEditing    EpisodeStatus = "editing"
	EpisodeStatusExported   EpisodeStatus = "exported"
	EpisodeStatusArchived   EpisodeStatus = "archived"

	WorkflowRunStatusDraft           WorkflowRunStatus = "draft"
	WorkflowRunStatusRunning         WorkflowRunStatus = "running"
	WorkflowRunStatusWaitingApproval WorkflowRunStatus = "waiting_approval"
	WorkflowRunStatusSucceeded       WorkflowRunStatus = "succeeded"
	WorkflowRunStatusFailed          WorkflowRunStatus = "failed"
	WorkflowRunStatusCanceled        WorkflowRunStatus = "canceled"

	WorkflowNodeRunStatusPending         WorkflowNodeRunStatus = "pending"
	WorkflowNodeRunStatusRunning         WorkflowNodeRunStatus = "running"
	WorkflowNodeRunStatusWaitingApproval WorkflowNodeRunStatus = "waiting_approval"
	WorkflowNodeRunStatusSucceeded       WorkflowNodeRunStatus = "succeeded"
	WorkflowNodeRunStatusFailed          WorkflowNodeRunStatus = "failed"
	WorkflowNodeRunStatusSkipped         WorkflowNodeRunStatus = "skipped"
	WorkflowNodeRunStatusCanceled        WorkflowNodeRunStatus = "canceled"

	AgentRunStatusPending   AgentRunStatus = "pending"
	AgentRunStatusRunning   AgentRunStatus = "running"
	AgentRunStatusSucceeded AgentRunStatus = "succeeded"
	AgentRunStatusFailed    AgentRunStatus = "failed"
	AgentRunStatusCanceled  AgentRunStatus = "canceled"
)

const (
	GenerationJobStatusDraft          GenerationJobStatus = "draft"
	GenerationJobStatusPreflight      GenerationJobStatus = "preflight"
	GenerationJobStatusQueued         GenerationJobStatus = "queued"
	GenerationJobStatusSubmitting     GenerationJobStatus = "submitting"
	GenerationJobStatusSubmitted      GenerationJobStatus = "submitted"
	GenerationJobStatusPolling        GenerationJobStatus = "polling"
	GenerationJobStatusDownloading    GenerationJobStatus = "downloading"
	GenerationJobStatusPostprocessing GenerationJobStatus = "postprocessing"
	GenerationJobStatusNeedsReview    GenerationJobStatus = "needs_review"
	GenerationJobStatusSucceeded      GenerationJobStatus = "succeeded"
	GenerationJobStatusBlocked        GenerationJobStatus = "blocked"
	GenerationJobStatusFailed         GenerationJobStatus = "failed"
	GenerationJobStatusTimedOut       GenerationJobStatus = "timed_out"
	GenerationJobStatusCanceling      GenerationJobStatus = "canceling"
	GenerationJobStatusCanceled       GenerationJobStatus = "canceled"

	AssetStatusDraft      AssetStatus = "draft"
	AssetStatusGenerating AssetStatus = "generating"
	AssetStatusReady      AssetStatus = "ready"
	AssetStatusFailed     AssetStatus = "failed"
	AssetStatusArchived   AssetStatus = "archived"

	ApprovalGateStatusPending          ApprovalGateStatus = "pending"
	ApprovalGateStatusApproved         ApprovalGateStatus = "approved"
	ApprovalGateStatusRejected         ApprovalGateStatus = "rejected"
	ApprovalGateStatusChangesRequested ApprovalGateStatus = "changes_requested"
	ApprovalGateStatusCanceled         ApprovalGateStatus = "canceled"

	TimelineStatusDraft     TimelineStatus = "draft"
	TimelineStatusSaved     TimelineStatus = "saved"
	TimelineStatusExporting TimelineStatus = "exporting"
	TimelineStatusExported  TimelineStatus = "exported"

	ExportStatusQueued    ExportStatus = "queued"
	ExportStatusRendering ExportStatus = "rendering"
	ExportStatusSucceeded ExportStatus = "succeeded"
	ExportStatusFailed    ExportStatus = "failed"
	ExportStatusCanceled  ExportStatus = "canceled"
)

func (s WorkflowRunStatus) CanTransitionTo(next WorkflowRunStatus) bool {
	return canTransition(workflowRunTransitions, string(s), string(next))
}

func (s WorkflowRunStatus) ValidateTransition(next WorkflowRunStatus) error {
	if s.CanTransitionTo(next) {
		return nil
	}
	return invalidTransition(string(s), string(next))
}

func (s GenerationJobStatus) CanTransitionTo(next GenerationJobStatus) bool {
	return canTransition(generationJobTransitions, string(s), string(next))
}

func (s GenerationJobStatus) ValidateTransition(next GenerationJobStatus) error {
	if s.CanTransitionTo(next) {
		return nil
	}
	return invalidTransition(string(s), string(next))
}

func (s ApprovalGateStatus) CanTransitionTo(next ApprovalGateStatus) bool {
	return canTransition(approvalGateTransitions, string(s), string(next))
}

func (s ApprovalGateStatus) ValidateTransition(next ApprovalGateStatus) error {
	if s.CanTransitionTo(next) {
		return nil
	}
	return invalidTransition(string(s), string(next))
}

func (s ExportStatus) CanTransitionTo(next ExportStatus) bool {
	return canTransition(exportTransitions, string(s), string(next))
}

func (s ExportStatus) ValidateTransition(next ExportStatus) error {
	if s.CanTransitionTo(next) {
		return nil
	}
	return invalidTransition(string(s), string(next))
}

func invalidTransition(current string, next string) error {
	return fmt.Errorf("%w: %s to %s", ErrInvalidTransition, current, next)
}
