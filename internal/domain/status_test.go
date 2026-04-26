package domain

import (
	"errors"
	"testing"
)

func TestWorkflowRunStatusTransition(t *testing.T) {
	t.Parallel()

	if err := WorkflowRunStatusDraft.ValidateTransition(WorkflowRunStatusRunning); err != nil {
		t.Fatalf("expected draft -> running to be valid: %v", err)
	}

	err := WorkflowRunStatusSucceeded.ValidateTransition(WorkflowRunStatusRunning)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
}

func TestGenerationJobStatusTransition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current GenerationJobStatus
		next    GenerationJobStatus
		wantErr bool
	}{
		{
			name:    "queued job can submit",
			current: GenerationJobStatusQueued,
			next:    GenerationJobStatusSubmitting,
		},
		{
			name:    "blocked job can re-enter preflight",
			current: GenerationJobStatusBlocked,
			next:    GenerationJobStatusPreflight,
		},
		{
			name:    "succeeded job cannot go back to polling",
			current: GenerationJobStatusSucceeded,
			next:    GenerationJobStatusPolling,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.current.ValidateTransition(tc.next)
			if tc.wantErr && !errors.Is(err, ErrInvalidTransition) {
				t.Fatalf("expected invalid transition, got %v", err)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected valid transition, got %v", err)
			}
		})
	}
}
