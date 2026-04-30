package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/workflow"
)

type storyAnalysisCheckpointStore struct {
	production repo.ProductionRepository
}

type persistedStoryAnalysisCheckpoint struct {
	WorkflowID string                                   `json:"workflow_id"`
	Sequence   uint64                                   `json:"sequence"`
	SavedAt    string                                   `json:"saved_at,omitempty"`
	Runs       map[string]persistedStoryAnalysisNodeRun `json:"runs"`
	Blackboard map[string]*AgentResult                  `json:"blackboard"`
}

type persistedStoryAnalysisNodeRun struct {
	NodeID string                 `json:"node_id"`
	Kind   workflow.NodeKind      `json:"kind"`
	Status workflow.NodeRunStatus `json:"status"`
	Output *AgentResult           `json:"output,omitempty"`
	Error  string                 `json:"error,omitempty"`
}

func newStoryAnalysisCheckpointStore(production repo.ProductionRepository) workflow.CheckpointStore {
	return &storyAnalysisCheckpointStore{production: production}
}

func (s *storyAnalysisCheckpointStore) Save(ctx context.Context, workflowID string, checkpoint *workflow.Checkpoint) error {
	payload, err := marshalStoryAnalysisCheckpoint(checkpoint)
	if err != nil {
		return err
	}
	return s.production.SaveWorkflowCheckpoint(ctx, workflowID, payload)
}

func (s *storyAnalysisCheckpointStore) Load(ctx context.Context, workflowID string) (*workflow.Checkpoint, error) {
	payload, err := s.production.LoadWorkflowCheckpoint(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if len(payload) == 0 || strings.TrimSpace(string(payload)) == "" || strings.TrimSpace(string(payload)) == "{}" {
		return nil, nil
	}
	return unmarshalStoryAnalysisCheckpoint(payload)
}

func marshalStoryAnalysisCheckpoint(checkpoint *workflow.Checkpoint) ([]byte, error) {
	if checkpoint == nil {
		return []byte("{}"), nil
	}
	persisted := persistedStoryAnalysisCheckpoint{
		WorkflowID: checkpoint.WorkflowID,
		Sequence:   checkpoint.Sequence,
		Runs:       make(map[string]persistedStoryAnalysisNodeRun, len(checkpoint.Runs)),
		Blackboard: make(map[string]*AgentResult, len(checkpoint.Blackboard)),
	}
	if !checkpoint.SavedAt.IsZero() {
		persisted.SavedAt = checkpoint.SavedAt.UTC().Format("2006-01-02T15:04:05.000000000Z07:00")
	}
	for nodeID, run := range checkpoint.Runs {
		output, err := agentResultFromValue(run.Output)
		if err != nil {
			return nil, err
		}
		persisted.Runs[nodeID] = persistedStoryAnalysisNodeRun{
			NodeID: run.NodeID,
			Kind:   run.Kind,
			Status: run.Status,
			Output: output,
			Error:  run.Error,
		}
	}
	for role, value := range checkpoint.Blackboard {
		result, err := agentResultFromValue(value)
		if err != nil {
			return nil, err
		}
		persisted.Blackboard[role] = result
	}
	return json.Marshal(persisted)
}

func unmarshalStoryAnalysisCheckpoint(payload []byte) (*workflow.Checkpoint, error) {
	var persisted persistedStoryAnalysisCheckpoint
	if err := json.Unmarshal(payload, &persisted); err != nil {
		return nil, err
	}
	checkpoint := &workflow.Checkpoint{
		WorkflowID: persisted.WorkflowID,
		Sequence:   persisted.Sequence,
		Runs:       make(map[string]workflow.NodeRunSnapshot, len(persisted.Runs)),
		Blackboard: make(map[string]any, len(persisted.Blackboard)),
	}
	if persisted.SavedAt != "" {
		if savedAt, err := time.Parse(time.RFC3339Nano, persisted.SavedAt); err == nil {
			checkpoint.SavedAt = savedAt
		}
	}
	for nodeID, run := range persisted.Runs {
		checkpoint.Runs[nodeID] = workflow.NodeRunSnapshot{
			NodeID: run.NodeID,
			Kind:   run.Kind,
			Status: run.Status,
			Output: cloneAgentResult(run.Output),
			Error:  run.Error,
		}
	}
	for role, result := range persisted.Blackboard {
		checkpoint.Blackboard[role] = cloneAgentResult(result)
	}
	return checkpoint, nil
}

func agentResultFromValue(value any) (*AgentResult, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case *AgentResult:
		return cloneAgentResult(typed), nil
	case AgentResult:
		return cloneAgentResult(&typed), nil
	default:
		payload, err := json.Marshal(typed)
		if err != nil {
			return nil, err
		}
		var result AgentResult
		if err := json.Unmarshal(payload, &result); err != nil {
			return nil, err
		}
		return cloneAgentResult(&result), nil
	}
}

func cloneAgentResult(result *AgentResult) *AgentResult {
	if result == nil {
		return nil
	}
	clone := *result
	clone.Highlights = append([]string(nil), result.Highlights...)
	return &clone
}
