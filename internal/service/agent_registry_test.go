package service

import (
	"strings"
	"testing"

	"github.com/yibaiba/dramora/internal/workflow"
)

func TestAgentRegistrySupportsPhase2Roles(t *testing.T) {
	t.Parallel()

	registry := mustNewAgentRegistry()
	for _, node := range workflow.Phase2Graph.Nodes {
		if !registry.HasRole(node.ID) {
			t.Fatalf("expected registry to include role %q", node.ID)
		}
	}
}

func TestAgentRegistryBuildUserPromptIncludesUpstreamContext(t *testing.T) {
	t.Parallel()

	registry := mustNewAgentRegistry()
	bb := workflow.NewBlackboard()
	bb.Write("outline_planner", &AgentResult{Role: "outline_planner", Output: "四段式大纲"})
	bb.Write("character_analyst", &AgentResult{Role: "character_analyst", Output: "角色清单"})
	bb.Write("scene_analyst", &AgentResult{Role: "scene_analyst", Output: "场景清单"})
	bb.Write("prop_analyst", &AgentResult{Role: "prop_analyst", Output: "道具清单"})

	prompt, err := registry.BuildUserPrompt("screenwriter", "雨夜故事", bb)
	if err != nil {
		t.Fatalf("BuildUserPrompt returned error: %v", err)
	}
	for _, expected := range []string{"雨夜故事", "四段式大纲", "角色清单", "场景清单", "道具清单"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected prompt to contain %q, got %q", expected, prompt)
		}
	}
}

func TestAgentRegistryRejectsUnknownRole(t *testing.T) {
	t.Parallel()

	registry := mustNewAgentRegistry()
	if _, err := registry.SystemPrompt("unknown_role"); err == nil {
		t.Fatal("expected unknown role to return error")
	}
	if _, err := registry.BuildUserPrompt("unknown_role", "x", workflow.NewBlackboard()); err == nil {
		t.Fatal("expected unknown role to return error")
	}
}
