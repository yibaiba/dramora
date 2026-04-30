package workflow

import (
	"context"
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func TestEnginePhase1DAG(t *testing.T) {
	bb := NewBlackboard()
	var callOrder []string
	var mu = make(chan string, 10)

	executor := func(_ context.Context, nodeID string, _ NodeKind, b *Blackboard) (any, error) {
		mu <- nodeID
		b.Write(nodeID, fmt.Sprintf("%s-output", nodeID))
		return fmt.Sprintf("%s-output", nodeID), nil
	}

	engine := NewEngine(Phase1Graph, bb, executor)
	if err := engine.Execute(context.Background()); err != nil {
		t.Fatalf("execute: %v", err)
	}
	close(mu)
	for nid := range mu {
		callOrder = append(callOrder, nid)
	}

	runs := engine.Runs()
	for _, n := range Phase1Graph.Nodes {
		run, ok := runs[n.ID]
		if !ok {
			t.Fatalf("missing run for %s", n.ID)
		}
		if run.Status != NodeSucceeded {
			t.Errorf("node %s status = %s, want succeeded", n.ID, run.Status)
		}
	}

	storyIdx := indexOf(callOrder, "story_analyst")
	outlineIdx := indexOf(callOrder, "outline_planner")
	if storyIdx >= outlineIdx {
		t.Error("story_analyst should run before outline_planner")
	}
	charIdx := indexOf(callOrder, "character_analyst")
	sceneIdx := indexOf(callOrder, "scene_analyst")
	propIdx := indexOf(callOrder, "prop_analyst")
	if outlineIdx >= charIdx || outlineIdx >= sceneIdx || outlineIdx >= propIdx {
		t.Error("outline_planner should run before character/scene/prop analysts")
	}
}

func TestEngineParallelExecution(t *testing.T) {
	bb := NewBlackboard()
	var concurrent int64
	var maxConcurrent int64

	executor := func(_ context.Context, nodeID string, _ NodeKind, b *Blackboard) (any, error) {
		cur := atomic.AddInt64(&concurrent, 1)
		for {
			old := atomic.LoadInt64(&maxConcurrent)
			if cur <= old || atomic.CompareAndSwapInt64(&maxConcurrent, old, cur) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt64(&concurrent, -1)
		b.Write(nodeID, "done")
		return "done", nil
	}

	engine := NewEngine(Phase1Graph, bb, executor)
	if err := engine.Execute(context.Background()); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if maxConcurrent < 2 {
		t.Errorf("expected parallel execution, max concurrent = %d", maxConcurrent)
	}
}

func TestEngineNodeFailureSkipsDependents(t *testing.T) {
	bb := NewBlackboard()
	executor := func(_ context.Context, nodeID string, _ NodeKind, b *Blackboard) (any, error) {
		if nodeID == "outline_planner" {
			return nil, fmt.Errorf("planner failed")
		}
		b.Write(nodeID, "ok")
		return "ok", nil
	}

	engine := NewEngine(Phase1Graph, bb, executor)
	_ = engine.Execute(context.Background())

	runs := engine.Runs()
	if runs["story_analyst"].Status != NodeSucceeded {
		t.Error("story_analyst should succeed")
	}
	if runs["outline_planner"].Status != NodeFailed {
		t.Error("outline_planner should fail")
	}
	for _, nid := range []string{"character_analyst", "scene_analyst", "prop_analyst"} {
		if runs[nid].Status != NodeSkipped {
			t.Errorf("%s should be skipped, got %s", nid, runs[nid].Status)
		}
	}
}

func TestBlackboardReadWrite(t *testing.T) {
	bb := NewBlackboard()
	bb.Write("story_analyst", map[string]string{"theme": "redemption"})

	val, ok := bb.Read("story_analyst")
	if !ok {
		t.Fatal("expected value")
	}
	m, _ := val.(map[string]string)
	if m["theme"] != "redemption" {
		t.Errorf("got %v", m)
	}

	_, ok = bb.Read("nonexistent")
	if ok {
		t.Error("expected no value")
	}
}

func TestBlackboardSubscribe(t *testing.T) {
	bb := NewBlackboard()
	ch := bb.Subscribe(5)

	bb.Write("agent_a", "result_a")
	bb.Write("agent_b", "result_b")

	select {
	case ev := <-ch:
		if ev.Role != "agent_a" {
			t.Errorf("expected agent_a, got %s", ev.Role)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
}

func TestEnginePhase2DAG(t *testing.T) {
	bb := NewBlackboard()
	order := make(chan string, 20)

	executor := func(_ context.Context, nodeID string, _ NodeKind, b *Blackboard) (any, error) {
		order <- nodeID
		b.Write(nodeID, nodeID+"-output")
		return nodeID + "-output", nil
	}

	engine := NewEngine(Phase2Graph, bb, executor)
	if err := engine.Execute(context.Background()); err != nil {
		t.Fatalf("execute: %v", err)
	}
	close(order)

	var callOrder []string
	for nid := range order {
		callOrder = append(callOrder, nid)
	}

	runs := engine.Runs()
	for _, n := range Phase2Graph.Nodes {
		run := runs[n.ID]
		if run.Status != NodeSucceeded {
			t.Errorf("node %s status = %s, want succeeded", n.ID, run.Status)
		}
	}

	// screenwriter must run after character/scene/prop
	swIdx := indexOf(callOrder, "screenwriter")
	for _, dep := range []string{"character_analyst", "scene_analyst", "prop_analyst"} {
		depIdx := indexOf(callOrder, dep)
		if depIdx >= swIdx {
			t.Errorf("%s should run before screenwriter", dep)
		}
	}

	// director/cinematographer/voice must run after screenwriter
	for _, child := range []string{"director", "cinematographer", "voice_subtitle"} {
		childIdx := indexOf(callOrder, child)
		if swIdx >= childIdx {
			t.Errorf("screenwriter should run before %s", child)
		}
	}
}

func TestEnginePhase2ParallelTiers(t *testing.T) {
	bb := NewBlackboard()
	var concurrent int64
	var maxConcurrent int64

	executor := func(_ context.Context, nodeID string, _ NodeKind, b *Blackboard) (any, error) {
		cur := atomic.AddInt64(&concurrent, 1)
		for {
			old := atomic.LoadInt64(&maxConcurrent)
			if cur <= old || atomic.CompareAndSwapInt64(&maxConcurrent, old, cur) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt64(&concurrent, -1)
		b.Write(nodeID, "done")
		return "done", nil
	}

	engine := NewEngine(Phase2Graph, bb, executor)
	if err := engine.Execute(context.Background()); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if maxConcurrent < 3 {
		t.Errorf("expected >=3 parallel nodes (C/S/P or D/C/V tier), max = %d", maxConcurrent)
	}
}

func TestEngineSavesCheckpointSnapshot(t *testing.T) {
	graph := &Graph{
		Nodes: []Node{
			{ID: "story", Kind: NodeKindStoryAnalysis},
			{ID: "outline", Kind: NodeKindStoryAnalysis},
		},
		Edges: []Edge{{FromNodeID: "story", ToNodeID: "outline"}},
	}
	bb := NewBlackboard()
	store := NewMemoryCheckpointStore()

	executor := func(_ context.Context, nodeID string, _ NodeKind, b *Blackboard) (any, error) {
		output := nodeID + "-output"
		b.Write(nodeID, output)
		return output, nil
	}

	engine := NewEngine(graph, bb, executor)
	engine.EnableCheckpointing("wf-1", store)
	if err := engine.Execute(context.Background()); err != nil {
		t.Fatalf("execute: %v", err)
	}

	checkpoint, err := store.Load(context.Background(), "wf-1")
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if checkpoint == nil {
		t.Fatal("expected checkpoint")
	}
	if checkpoint.Sequence == 0 {
		t.Fatal("expected checkpoint sequence to advance")
	}
	if checkpoint.Runs["story"].Status != NodeSucceeded {
		t.Fatalf("story status = %s, want succeeded", checkpoint.Runs["story"].Status)
	}
	if checkpoint.Runs["outline"].Status != NodeSucceeded {
		t.Fatalf("outline status = %s, want succeeded", checkpoint.Runs["outline"].Status)
	}
	if got := checkpoint.Blackboard["story"]; got != "story-output" {
		t.Fatalf("story blackboard = %v, want story-output", got)
	}
}

func TestEngineResumeSkipsSucceededNodesAndRestoresBlackboard(t *testing.T) {
	graph := &Graph{
		Nodes: []Node{
			{ID: "story", Kind: NodeKindStoryAnalysis},
			{ID: "outline", Kind: NodeKindStoryAnalysis},
		},
		Edges: []Edge{{FromNodeID: "story", ToNodeID: "outline"}},
	}
	bb := NewBlackboard()
	var calls []string

	executor := func(_ context.Context, nodeID string, _ NodeKind, b *Blackboard) (any, error) {
		calls = append(calls, nodeID)
		if nodeID == "outline" {
			story, ok := b.Read("story")
			if !ok || story != "story-output" {
				t.Fatalf("outline expected restored story output, got %v, ok=%v", story, ok)
			}
		}
		output := nodeID + "-output"
		b.Write(nodeID, output)
		return output, nil
	}

	engine := NewEngine(graph, bb, executor)
	if err := engine.Resume(&Checkpoint{
		WorkflowID: "wf-2",
		Sequence:   3,
		Runs: map[string]NodeRunSnapshot{
			"story": {
				NodeID: "story",
				Kind:   NodeKindStoryAnalysis,
				Status: NodeSucceeded,
				Output: "story-output",
			},
			"outline": {
				NodeID: "outline",
				Kind:   NodeKindStoryAnalysis,
				Status: NodeWaiting,
			},
		},
		Blackboard: map[string]any{
			"story": "story-output",
		},
	}); err != nil {
		t.Fatalf("resume: %v", err)
	}

	if err := engine.Execute(context.Background()); err != nil {
		t.Fatalf("execute resumed engine: %v", err)
	}
	if !reflect.DeepEqual(calls, []string{"outline"}) {
		t.Fatalf("calls = %v, want [outline]", calls)
	}
	runs := engine.Runs()
	if runs["story"].Status != NodeSucceeded {
		t.Fatalf("story status = %s, want succeeded", runs["story"].Status)
	}
	if runs["outline"].Status != NodeSucceeded {
		t.Fatalf("outline status = %s, want succeeded", runs["outline"].Status)
	}
}

func TestEngineResumeRequeuesRunningNodes(t *testing.T) {
	graph := &Graph{
		Nodes: []Node{
			{ID: "story", Kind: NodeKindStoryAnalysis},
			{ID: "outline", Kind: NodeKindStoryAnalysis},
		},
		Edges: []Edge{{FromNodeID: "story", ToNodeID: "outline"}},
	}
	bb := NewBlackboard()
	var calls []string

	executor := func(_ context.Context, nodeID string, _ NodeKind, b *Blackboard) (any, error) {
		calls = append(calls, nodeID)
		b.Write(nodeID, nodeID+"-output")
		return nodeID + "-output", nil
	}

	engine := NewEngine(graph, bb, executor)
	if err := engine.Resume(&Checkpoint{
		WorkflowID: "wf-3",
		Sequence:   2,
		Runs: map[string]NodeRunSnapshot{
			"story": {
				NodeID: "story",
				Kind:   NodeKindStoryAnalysis,
				Status: NodeRunning,
			},
			"outline": {
				NodeID: "outline",
				Kind:   NodeKindStoryAnalysis,
				Status: NodeWaiting,
			},
		},
	}); err != nil {
		t.Fatalf("resume: %v", err)
	}

	if err := engine.Execute(context.Background()); err != nil {
		t.Fatalf("execute resumed engine: %v", err)
	}
	if !reflect.DeepEqual(calls, []string{"story", "outline"}) {
		t.Fatalf("calls = %v, want [story outline]", calls)
	}
}

func TestEngineResumeIsIdempotentAfterCompletion(t *testing.T) {
	graph := &Graph{
		Nodes: []Node{
			{ID: "story", Kind: NodeKindStoryAnalysis},
			{ID: "outline", Kind: NodeKindStoryAnalysis},
		},
		Edges: []Edge{{FromNodeID: "story", ToNodeID: "outline"}},
	}
	store := NewMemoryCheckpointStore()
	var calls []string

	executor := func(_ context.Context, nodeID string, _ NodeKind, b *Blackboard) (any, error) {
		calls = append(calls, nodeID)
		b.Write(nodeID, nodeID+"-output")
		return nodeID + "-output", nil
	}

	engine := NewEngine(graph, NewBlackboard(), executor)
	engine.EnableCheckpointing("wf-4", store)
	if err := engine.Resume(&Checkpoint{
		WorkflowID: "wf-4",
		Sequence:   1,
		Runs: map[string]NodeRunSnapshot{
			"story": {
				NodeID: "story",
				Kind:   NodeKindStoryAnalysis,
				Status: NodeSucceeded,
				Output: "story-output",
			},
			"outline": {
				NodeID: "outline",
				Kind:   NodeKindStoryAnalysis,
				Status: NodeWaiting,
			},
		},
		Blackboard: map[string]any{
			"story": "story-output",
		},
	}); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if err := engine.Execute(context.Background()); err != nil {
		t.Fatalf("execute resumed engine: %v", err)
	}

	checkpoint, err := store.Load(context.Background(), "wf-4")
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}

	resumed := NewEngine(graph, NewBlackboard(), executor)
	if err := resumed.Resume(checkpoint); err != nil {
		t.Fatalf("resume completed checkpoint: %v", err)
	}
	if err := resumed.Execute(context.Background()); err != nil {
		t.Fatalf("execute completed checkpoint: %v", err)
	}

	if !reflect.DeepEqual(calls, []string{"outline"}) {
		t.Fatalf("calls = %v, want [outline]", calls)
	}
}

func indexOf(slice []string, val string) int {
	for i, v := range slice {
		if v == val {
			return i
		}
	}
	return -1
}
