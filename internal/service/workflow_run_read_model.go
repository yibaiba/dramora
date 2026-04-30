package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/workflow"
)

type WorkflowCheckpointSummary struct {
	Sequence        uint64
	SavedAt         time.Time
	CompletedNodes  int
	WaitingNodes    int
	RunningNodes    int
	FailedNodes     int
	SkippedNodes    int
	BlackboardRoles []string
}

type WorkflowRunDetail struct {
	Run        domain.WorkflowRun
	Checkpoint *WorkflowCheckpointSummary
	NodeRuns   []WorkflowNodeDetail
}

type WorkflowNodeDetail struct {
	NodeID          string
	Kind            workflow.NodeKind
	Status          domain.WorkflowNodeRunStatus
	Summary         string
	Highlights      []string
	ErrorMessage    string
	UpstreamNodeIDs []string
}

func (s *ProductionService) GetWorkflowRunDetail(ctx context.Context, id string) (WorkflowRunDetail, error) {
	if strings.TrimSpace(id) == "" {
		return WorkflowRunDetail{}, fmt.Errorf("%w: workflow run id is required", domain.ErrInvalidInput)
	}
	run, err := s.GetWorkflowRun(ctx, id)
	if err != nil {
		return WorkflowRunDetail{}, err
	}
	checkpointStore := newStoryAnalysisCheckpointStore(s.production)
	checkpoint, err := checkpointStore.Load(ctx, id)
	if err != nil {
		return WorkflowRunDetail{}, err
	}
	checkpoint = normalizeCheckpointForRun(run, checkpoint)
	return WorkflowRunDetail{
		Run:        run,
		Checkpoint: buildWorkflowCheckpointSummary(checkpoint),
		NodeRuns:   buildWorkflowNodeDetails(checkpoint),
	}, nil
}

func buildWorkflowCheckpointSummary(checkpoint *workflow.Checkpoint) *WorkflowCheckpointSummary {
	if checkpoint == nil {
		return nil
	}
	summary := &WorkflowCheckpointSummary{
		Sequence:        checkpoint.Sequence,
		SavedAt:         checkpoint.SavedAt,
		BlackboardRoles: make([]string, 0, len(checkpoint.Blackboard)),
	}
	for _, run := range checkpoint.Runs {
		switch run.Status {
		case workflow.NodeSucceeded:
			summary.CompletedNodes++
		case workflow.NodeWaiting:
			summary.WaitingNodes++
		case workflow.NodeRunning:
			summary.RunningNodes++
		case workflow.NodeFailed:
			summary.FailedNodes++
		case workflow.NodeSkipped:
			summary.SkippedNodes++
		}
	}
	for role := range checkpoint.Blackboard {
		summary.BlackboardRoles = append(summary.BlackboardRoles, role)
	}
	sort.Strings(summary.BlackboardRoles)
	return summary
}

func buildWorkflowNodeDetails(checkpoint *workflow.Checkpoint) []WorkflowNodeDetail {
	if checkpoint == nil || len(checkpoint.Runs) == 0 {
		return nil
	}

	detailsByID := make(map[string]WorkflowNodeDetail, len(checkpoint.Runs))
	for nodeID, run := range checkpoint.Runs {
		resolvedNodeID := run.NodeID
		if strings.TrimSpace(resolvedNodeID) == "" {
			resolvedNodeID = nodeID
		}
		detail := WorkflowNodeDetail{
			NodeID:       resolvedNodeID,
			Kind:         run.Kind,
			Status:       mapWorkflowNodeStatus(run.Status),
			ErrorMessage: strings.TrimSpace(run.Error),
		}
		if result, err := agentResultFromValue(run.Output); err == nil && result != nil {
			detail.Highlights = append([]string(nil), result.Highlights...)
			detail.Summary = summarizeWorkflowNodeOutput(result.Output, result.Highlights)
		}
		detailsByID[resolvedNodeID] = detail
	}

	orderedIDs := orderedWorkflowNodeIDs(detailsByID)
	details := make([]WorkflowNodeDetail, 0, len(detailsByID))
	for _, nodeID := range orderedIDs {
		detail, ok := detailsByID[nodeID]
		if !ok {
			continue
		}
		detail.UpstreamNodeIDs = workflowNodeUpstreamIDs(nodeID)
		details = append(details, detail)
	}
	return details
}

func normalizeCheckpointForRun(run domain.WorkflowRun, checkpoint *workflow.Checkpoint) *workflow.Checkpoint {
	if checkpoint == nil || run.Status != domain.WorkflowRunStatusSucceeded {
		return checkpoint
	}

	normalized := checkpoint.Clone()
	if normalized.Runs == nil {
		normalized.Runs = make(map[string]workflow.NodeRunSnapshot, len(workflow.Phase1Graph.Nodes))
	}
	for _, node := range workflow.Phase1Graph.Nodes {
		snapshot, ok := normalized.Runs[node.ID]
		if !ok {
			normalized.Runs[node.ID] = workflow.NodeRunSnapshot{
				NodeID: node.ID,
				Kind:   node.Kind,
				Status: workflow.NodeSucceeded,
			}
			continue
		}
		if snapshot.NodeID == "" {
			snapshot.NodeID = node.ID
		}
		if snapshot.Kind == "" {
			snapshot.Kind = node.Kind
		}
		switch snapshot.Status {
		case workflow.NodeRunning, workflow.NodeWaiting:
			snapshot.Status = workflow.NodeSucceeded
		}
		normalized.Runs[node.ID] = snapshot
	}
	return normalized
}

func orderedWorkflowNodeIDs(detailsByID map[string]WorkflowNodeDetail) []string {
	ordered := make([]string, 0, len(detailsByID))
	seen := make(map[string]struct{}, len(detailsByID))
	for _, node := range workflow.Phase1Graph.Nodes {
		if _, ok := detailsByID[node.ID]; !ok {
			continue
		}
		ordered = append(ordered, node.ID)
		seen[node.ID] = struct{}{}
	}

	extra := make([]string, 0, len(detailsByID)-len(ordered))
	for nodeID := range detailsByID {
		if _, ok := seen[nodeID]; ok {
			continue
		}
		extra = append(extra, nodeID)
	}
	sort.Strings(extra)
	return append(ordered, extra...)
}

func workflowNodeUpstreamIDs(nodeID string) []string {
	dependencies := make([]string, 0, len(workflow.Phase1Graph.Edges))
	for _, edge := range workflow.Phase1Graph.Edges {
		if edge.ToNodeID != nodeID {
			continue
		}
		dependencies = append(dependencies, edge.FromNodeID)
	}
	return dependencies
}

func mapWorkflowNodeStatus(status workflow.NodeRunStatus) domain.WorkflowNodeRunStatus {
	switch status {
	case workflow.NodeRunning:
		return domain.WorkflowNodeRunStatusRunning
	case workflow.NodeSucceeded:
		return domain.WorkflowNodeRunStatusSucceeded
	case workflow.NodeFailed:
		return domain.WorkflowNodeRunStatusFailed
	case workflow.NodeSkipped:
		return domain.WorkflowNodeRunStatusSkipped
	default:
		return domain.WorkflowNodeRunStatusPending
	}
}

func summarizeWorkflowNodeOutput(output string, highlights []string) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(output)), " ")
	if normalized == "" && len(highlights) > 0 {
		normalized = strings.Join(highlights[:min(2, len(highlights))], " · ")
	}
	if utf8.RuneCountInString(normalized) <= 140 {
		return normalized
	}
	runes := []rune(normalized)
	return string(runes[:140]) + "…"
}

func min(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
