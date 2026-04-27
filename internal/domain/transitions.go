package domain

var workflowRunTransitions = map[string][]string{
	string(WorkflowRunStatusDraft): {
		string(WorkflowRunStatusRunning),
		string(WorkflowRunStatusCanceled),
	},
	string(WorkflowRunStatusRunning): {
		string(WorkflowRunStatusWaitingApproval),
		string(WorkflowRunStatusSucceeded),
		string(WorkflowRunStatusFailed),
		string(WorkflowRunStatusCanceled),
	},
	string(WorkflowRunStatusWaitingApproval): {
		string(WorkflowRunStatusRunning),
		string(WorkflowRunStatusFailed),
		string(WorkflowRunStatusCanceled),
	},
}

var generationJobTransitions = map[string][]string{
	string(GenerationJobStatusDraft): {
		string(GenerationJobStatusPreflight),
		string(GenerationJobStatusCanceled),
	},
	string(GenerationJobStatusPreflight): {
		string(GenerationJobStatusQueued),
		string(GenerationJobStatusBlocked),
		string(GenerationJobStatusFailed),
	},
	string(GenerationJobStatusQueued): {
		string(GenerationJobStatusSubmitting),
		string(GenerationJobStatusCanceling),
	},
	string(GenerationJobStatusSubmitting): {
		string(GenerationJobStatusSubmitted),
		string(GenerationJobStatusFailed),
		string(GenerationJobStatusTimedOut),
	},
	string(GenerationJobStatusSubmitted): {
		string(GenerationJobStatusPolling),
		string(GenerationJobStatusDownloading),
		string(GenerationJobStatusFailed),
		string(GenerationJobStatusTimedOut),
	},
	string(GenerationJobStatusPolling): {
		string(GenerationJobStatusDownloading),
		string(GenerationJobStatusFailed),
		string(GenerationJobStatusTimedOut),
		string(GenerationJobStatusCanceling),
	},
	string(GenerationJobStatusDownloading): {
		string(GenerationJobStatusPostprocessing),
		string(GenerationJobStatusFailed),
	},
	string(GenerationJobStatusPostprocessing): {
		string(GenerationJobStatusNeedsReview),
		string(GenerationJobStatusSucceeded),
		string(GenerationJobStatusFailed),
	},
	string(GenerationJobStatusNeedsReview): {
		string(GenerationJobStatusSucceeded),
		string(GenerationJobStatusBlocked),
		string(GenerationJobStatusFailed),
	},
	string(GenerationJobStatusBlocked): {
		string(GenerationJobStatusPreflight),
		string(GenerationJobStatusCanceled),
	},
	string(GenerationJobStatusCanceling): {
		string(GenerationJobStatusCanceled),
		string(GenerationJobStatusFailed),
	},
}

var approvalGateTransitions = map[string][]string{
	string(ApprovalGateStatusPending): {
		string(ApprovalGateStatusApproved),
		string(ApprovalGateStatusRejected),
		string(ApprovalGateStatusChangesRequested),
		string(ApprovalGateStatusCanceled),
	},
	string(ApprovalGateStatusChangesRequested): {
		string(ApprovalGateStatusPending),
		string(ApprovalGateStatusCanceled),
	},
}

var exportTransitions = map[string][]string{
	string(ExportStatusQueued): {
		string(ExportStatusRendering),
		string(ExportStatusCanceled),
	},
	string(ExportStatusRendering): {
		string(ExportStatusSucceeded),
		string(ExportStatusFailed),
		string(ExportStatusCanceled),
	},
}

func canTransition(transitions map[string][]string, current string, next string) bool {
	if current == next {
		return true
	}
	for _, allowed := range transitions[current] {
		if allowed == next {
			return true
		}
	}
	return false
}
