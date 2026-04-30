package workflow

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type NodeRunStatus string

const (
	NodeWaiting   NodeRunStatus = "waiting"
	NodeRunning   NodeRunStatus = "running"
	NodeSucceeded NodeRunStatus = "succeeded"
	NodeFailed    NodeRunStatus = "failed"
	NodeSkipped   NodeRunStatus = "skipped"
)

type NodeRun struct {
	NodeID string
	Kind   NodeKind
	Status NodeRunStatus
	Output any
	Error  error
}

type NodeExecutor func(ctx context.Context, nodeID string, kind NodeKind, bb *Blackboard) (any, error)

type Engine struct {
	graph           *Graph
	bb              *Blackboard
	executor        NodeExecutor
	mu              sync.Mutex
	runs            map[string]*NodeRun
	checkpointStore CheckpointStore
	checkpointKey   string
	checkpointSeq   uint64
	executionErr    error
}

func NewEngine(graph *Graph, bb *Blackboard, executor NodeExecutor) *Engine {
	runs := make(map[string]*NodeRun, len(graph.Nodes))
	for _, n := range graph.Nodes {
		runs[n.ID] = &NodeRun{
			NodeID: n.ID,
			Kind:   n.Kind,
			Status: NodeWaiting,
		}
	}
	return &Engine{graph: graph, bb: bb, executor: executor, runs: runs}
}

func (e *Engine) EnableCheckpointing(workflowID string, store CheckpointStore) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.checkpointKey = workflowID
	e.checkpointStore = store
}

func (e *Engine) Resume(checkpoint *Checkpoint) error {
	if checkpoint == nil {
		return nil
	}
	if checkpoint.WorkflowID != "" && e.checkpointKey != "" && checkpoint.WorkflowID != e.checkpointKey {
		return fmt.Errorf("checkpoint workflow id mismatch: %s != %s", checkpoint.WorkflowID, e.checkpointKey)
	}

	for nodeID, run := range checkpoint.Runs {
		current, ok := e.runs[nodeID]
		if !ok {
			return fmt.Errorf("checkpoint contains unknown node %q", nodeID)
		}
		if run.Kind != "" && run.Kind != current.Kind {
			return fmt.Errorf("checkpoint kind mismatch for node %q", nodeID)
		}
	}

	e.bb.Restore(checkpoint.Blackboard)

	e.mu.Lock()
	defer e.mu.Unlock()
	for _, node := range e.graph.Nodes {
		run := e.runs[node.ID]
		snapshot, ok := checkpoint.Runs[node.ID]
		if !ok {
			run.Status = NodeWaiting
			run.Output = nil
			run.Error = nil
			continue
		}
		run.Status = snapshot.Status
		if run.Status == NodeRunning {
			run.Status = NodeWaiting
		}
		if run.Status == "" {
			run.Status = NodeWaiting
		}
		run.Output = snapshot.Output
		run.Error = nil
		if snapshot.Error != "" {
			run.Error = errors.New(snapshot.Error)
		}
	}
	if checkpoint.WorkflowID != "" {
		e.checkpointKey = checkpoint.WorkflowID
	}
	if checkpoint.Sequence > e.checkpointSeq {
		e.checkpointSeq = checkpoint.Sequence
	}
	return nil
}

func (e *Engine) Execute(ctx context.Context) error {
	e.clearExecutionError()
	order, err := e.topoSort()
	if err != nil {
		return err
	}
	deps := e.buildDeps()
	if err := e.saveCheckpoint(ctx); err != nil {
		return err
	}

	for len(order) > 0 {
		ready := e.findReady(ctx, order, deps)
		if len(ready) == 0 {
			break
		}

		var wg sync.WaitGroup
		for _, nodeID := range ready {
			wg.Add(1)
			go func(nid string) {
				defer wg.Done()
				e.executeNode(ctx, nid)
			}(nodeID)
		}
		wg.Wait()
		if err := e.executionErrorValue(); err != nil {
			return err
		}

		remaining := make([]string, 0, len(order))
		for _, nid := range order {
			run := e.getRun(nid)
			if run.Status == NodeWaiting || run.Status == NodeRunning {
				remaining = append(remaining, nid)
			}
		}
		order = remaining
	}
	if err := e.executionErrorValue(); err != nil {
		return err
	}
	return nil
}

func (e *Engine) Runs() map[string]*NodeRun {
	e.mu.Lock()
	defer e.mu.Unlock()
	cp := make(map[string]*NodeRun, len(e.runs))
	for k, v := range e.runs {
		cp[k] = v
	}
	return cp
}

func (e *Engine) executeNode(ctx context.Context, nodeID string) {
	if err := e.setStatus(ctx, nodeID, NodeRunning); err != nil {
		e.recordExecutionError(err)
		return
	}
	output, err := e.executor(ctx, nodeID, e.getRun(nodeID).Kind, e.bb)
	if err != nil {
		if saveErr := e.setResult(ctx, nodeID, NodeFailed, nil, err); saveErr != nil {
			e.recordExecutionError(saveErr)
		}
		return
	}
	if saveErr := e.setResult(ctx, nodeID, NodeSucceeded, output, nil); saveErr != nil {
		e.recordExecutionError(saveErr)
	}
}

func (e *Engine) findReady(ctx context.Context, order []string, deps map[string][]string) []string {
	e.mu.Lock()
	var ready []string
	var skippedChanged bool
	for _, nid := range order {
		run := e.runs[nid]
		if run.Status != NodeWaiting {
			continue
		}
		allDepsOK := true
		for _, dep := range deps[nid] {
			depRun := e.runs[dep]
			if depRun.Status == NodeSucceeded {
				continue
			}
			if depRun.Status == NodeFailed || depRun.Status == NodeSkipped {
				run.Status = NodeSkipped
				run.Error = nil
				run.Output = nil
				allDepsOK = false
				skippedChanged = true
				break
			}
			allDepsOK = false
		}
		if allDepsOK && run.Status == NodeWaiting {
			ready = append(ready, nid)
		}
	}
	e.mu.Unlock()
	if skippedChanged {
		if err := e.saveCheckpoint(ctx); err != nil {
			e.recordExecutionError(err)
		}
	}
	return ready
}

func (e *Engine) setStatus(ctx context.Context, nodeID string, s NodeRunStatus) error {
	e.mu.Lock()
	e.runs[nodeID].Status = s
	e.runs[nodeID].Error = nil
	if s != NodeSucceeded {
		e.runs[nodeID].Output = nil
	}
	e.mu.Unlock()
	return e.saveCheckpoint(ctx)
}

func (e *Engine) setResult(ctx context.Context, nodeID string, status NodeRunStatus, output any, err error) error {
	e.mu.Lock()
	run := e.runs[nodeID]
	run.Status = status
	run.Output = output
	run.Error = err
	e.mu.Unlock()
	return e.saveCheckpoint(ctx)
}

func (e *Engine) getRun(nodeID string) *NodeRun {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.runs[nodeID]
}

func (e *Engine) buildDeps() map[string][]string {
	deps := make(map[string][]string)
	for _, edge := range e.graph.Edges {
		deps[edge.ToNodeID] = append(deps[edge.ToNodeID], edge.FromNodeID)
	}
	return deps
}

func (e *Engine) saveCheckpoint(ctx context.Context) error {
	e.mu.Lock()
	store := e.checkpointStore
	workflowID := e.checkpointKey
	if store == nil || workflowID == "" {
		e.mu.Unlock()
		return nil
	}
	checkpoint := e.buildCheckpointLocked()
	e.mu.Unlock()
	return store.Save(ctx, workflowID, checkpoint)
}

func (e *Engine) buildCheckpointLocked() *Checkpoint {
	e.checkpointSeq++
	checkpoint := &Checkpoint{
		WorkflowID: e.checkpointKey,
		Sequence:   e.checkpointSeq,
		SavedAt:    time.Now().UTC(),
		Runs:       make(map[string]NodeRunSnapshot, len(e.runs)),
		Blackboard: make(map[string]any),
	}
	e.bb.mu.RLock()
	for key, value := range e.bb.state {
		checkpoint.Blackboard[key] = value
	}
	e.bb.mu.RUnlock()
	for nodeID, run := range e.runs {
		snapshot := NodeRunSnapshot{
			NodeID: run.NodeID,
			Kind:   run.Kind,
			Status: run.Status,
			Output: run.Output,
		}
		if run.Error != nil {
			snapshot.Error = run.Error.Error()
		}
		checkpoint.Runs[nodeID] = snapshot
	}
	return checkpoint
}

func (e *Engine) clearExecutionError() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.executionErr = nil
}

func (e *Engine) recordExecutionError(err error) {
	if err == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.executionErr == nil {
		e.executionErr = err
	}
}

func (e *Engine) executionErrorValue() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.executionErr
}

func (e *Engine) topoSort() ([]string, error) {
	inDegree := make(map[string]int)
	adj := make(map[string][]string)
	for _, n := range e.graph.Nodes {
		inDegree[n.ID] = 0
	}
	for _, edge := range e.graph.Edges {
		adj[edge.FromNodeID] = append(adj[edge.FromNodeID], edge.ToNodeID)
		inDegree[edge.ToNodeID]++
	}

	var queue []string
	for _, n := range e.graph.Nodes {
		if inDegree[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}

	var order []string
	for len(queue) > 0 {
		nid := queue[0]
		queue = queue[1:]
		order = append(order, nid)
		for _, next := range adj[nid] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(order) != len(e.graph.Nodes) {
		return nil, fmt.Errorf("cycle detected in DAG")
	}
	return order, nil
}
