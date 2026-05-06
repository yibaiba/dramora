package service

import (
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/yibaiba/dramora/internal/workflow"
)

//go:embed promptpacks/agents/*
var agentPromptFiles embed.FS

type AgentDefinition struct {
	Role         string
	SystemPrompt string
	userTemplate *template.Template
}

type AgentRegistry struct {
	definitions map[string]AgentDefinition
}

type agentPromptTemplateData struct {
	SourceText             string
	StoryAnalystOutput     string
	OutlinePlannerOutput   string
	CharacterAnalystOutput string
	SceneAnalystOutput     string
	PropAnalystOutput      string
	ScreenwriterOutput     string
}

func mustNewAgentRegistry() *AgentRegistry {
	registry, err := newAgentRegistry()
	if err != nil {
		panic(err)
	}
	return registry
}

func newAgentRegistry() (*AgentRegistry, error) {
	roles := []string{
		"story_analyst",
		"outline_planner",
		"character_analyst",
		"scene_analyst",
		"prop_analyst",
		"screenwriter",
		"director",
		"cinematographer",
		"voice_subtitle",
	}

	definitions := make(map[string]AgentDefinition, len(roles))
	for _, role := range roles {
		systemPrompt, err := readEmbeddedPrompt("promptpacks/agents/" + role + ".system.md")
		if err != nil {
			return nil, err
		}
		userTemplate, err := readEmbeddedTemplate(role, "promptpacks/agents/"+role+".user.tmpl")
		if err != nil {
			return nil, err
		}
		definitions[role] = AgentDefinition{
			Role:         role,
			SystemPrompt: systemPrompt,
			userTemplate: userTemplate,
		}
	}

	return &AgentRegistry{definitions: definitions}, nil
}

func readEmbeddedPrompt(path string) (string, error) {
	content, err := agentPromptFiles.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read prompt %s: %w", path, err)
	}
	return strings.TrimSpace(string(content)), nil
}

func readEmbeddedTemplate(role string, path string) (*template.Template, error) {
	content, err := agentPromptFiles.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read prompt template %s: %w", path, err)
	}
	tmpl, err := template.New(role).Option("missingkey=error").Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse prompt template %s: %w", path, err)
	}
	return tmpl, nil
}

func (r *AgentRegistry) HasRole(role string) bool {
	if r == nil {
		return false
	}
	_, ok := r.definitions[role]
	return ok
}

func (r *AgentRegistry) SystemPrompt(role string) (string, error) {
	definition, ok := r.definitions[role]
	if !ok {
		return "", fmt.Errorf("unsupported agent role: %s", role)
	}
	return definition.SystemPrompt, nil
}

func (r *AgentRegistry) BuildUserPrompt(role string, sourceText string, bb *workflow.Blackboard) (string, error) {
	definition, ok := r.definitions[role]
	if !ok {
		return "", fmt.Errorf("unsupported agent role: %s", role)
	}

	data := buildAgentPromptTemplateData(sourceText, bb)
	var rendered strings.Builder
	if err := definition.userTemplate.Execute(&rendered, data); err != nil {
		return "", fmt.Errorf("render %s prompt: %w", role, err)
	}
	return strings.TrimSpace(rendered.String()), nil
}

func buildAgentPromptTemplateData(sourceText string, bb *workflow.Blackboard) agentPromptTemplateData {
	return agentPromptTemplateData{
		SourceText:             sourceText,
		StoryAnalystOutput:     blackboardAgentOutput(bb, "story_analyst"),
		OutlinePlannerOutput:   blackboardAgentOutput(bb, "outline_planner"),
		CharacterAnalystOutput: blackboardAgentOutput(bb, "character_analyst"),
		SceneAnalystOutput:     blackboardAgentOutput(bb, "scene_analyst"),
		PropAnalystOutput:      blackboardAgentOutput(bb, "prop_analyst"),
		ScreenwriterOutput:     blackboardAgentOutput(bb, "screenwriter"),
	}
}

func blackboardAgentOutput(bb *workflow.Blackboard, role string) string {
	if bb == nil {
		return ""
	}
	value, ok := bb.Read(role)
	if !ok {
		return ""
	}

	switch typed := value.(type) {
	case *AgentResult:
		return typed.Output
	case AgentResult:
		return typed.Output
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}
