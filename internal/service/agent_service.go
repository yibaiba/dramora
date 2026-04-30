package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/yibaiba/dramora/internal/provider"
	"github.com/yibaiba/dramora/internal/workflow"
)

type AgentService struct {
	providerSvc      *ProviderService
	executorFactory  func(sourceText string) workflow.NodeExecutor
	availabilityFunc func(ctx context.Context) bool
}

func NewAgentService(providerSvc *ProviderService) *AgentService {
	return &AgentService{providerSvc: providerSvc}
}

type AgentResult struct {
	Role        string
	Output      string
	Highlights  []string
	TokenCount  int
	DurationMS  int64
	RawResponse string
}

func (s *AgentService) MakeNodeExecutor(sourceText string) workflow.NodeExecutor {
	if s != nil && s.executorFactory != nil {
		return s.executorFactory(sourceText)
	}
	return func(ctx context.Context, nodeID string, kind workflow.NodeKind, bb *workflow.Blackboard) (any, error) {
		prompt := buildAgentPrompt(nodeID, sourceText, bb)
		result, err := s.callLLM(ctx, nodeID, prompt)
		if err != nil {
			return nil, err
		}
		bb.Write(nodeID, result)
		return result, nil
	}
}

func (s *AgentService) callLLM(ctx context.Context, role string, prompt string) (*AgentResult, error) {
	cfg, err := s.providerSvc.GetProviderConfig(ctx, "chat")
	if err != nil {
		return nil, fmt.Errorf("chat 端点未配置: %w", err)
	}

	client := provider.NewChatClient(
		cfg.BaseURL, cfg.APIKey, cfg.Model,
		time.Duration(cfg.TimeoutMS)*time.Millisecond,
	)

	start := time.Now()
	resp, err := client.Complete(ctx, []provider.ChatMessage{
		{Role: "system", Content: systemPromptForRole(role)},
		{Role: "user", Content: prompt},
	})
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return nil, err
	}

	content := resp.Content()
	highlights := extractHighlights(role, content)

	return &AgentResult{
		Role:        role,
		Output:      content,
		Highlights:  highlights,
		TokenCount:  resp.Usage.TotalTokens,
		DurationMS:  elapsed,
		RawResponse: content,
	}, nil
}

func (s *AgentService) IsAvailable(ctx context.Context) bool {
	if s != nil && s.availabilityFunc != nil {
		return s.availabilityFunc(ctx)
	}
	_, err := s.providerSvc.GetProviderConfig(ctx, "chat")
	return err == nil
}

func buildAgentPrompt(role string, sourceText string, bb *workflow.Blackboard) string {
	var sb strings.Builder

	switch role {
	case "story_analyst":
		sb.WriteString(storyAnalystUserPrompt(sourceText))
	case "outline_planner":
		prev, _ := bb.Read("story_analyst")
		sb.WriteString(outlinePlannerUserPrompt(sourceText, prev))
	case "character_analyst":
		prev, _ := bb.Read("outline_planner")
		sb.WriteString(characterAnalystUserPrompt(sourceText, prev))
	case "scene_analyst":
		prev, _ := bb.Read("outline_planner")
		sb.WriteString(sceneAnalystUserPrompt(sourceText, prev))
	case "prop_analyst":
		prev, _ := bb.Read("outline_planner")
		sb.WriteString(propAnalystUserPrompt(sourceText, prev))
	case "screenwriter":
		sb.WriteString(screenwriterUserPrompt(sourceText, bb))
	case "director":
		prev, _ := bb.Read("screenwriter")
		sb.WriteString(directorUserPrompt(sourceText, prev))
	case "cinematographer":
		prev, _ := bb.Read("screenwriter")
		sb.WriteString(cinematographerUserPrompt(sourceText, prev))
	case "voice_subtitle":
		prev, _ := bb.Read("screenwriter")
		sb.WriteString(voiceSubtitleUserPrompt(sourceText, prev))
	}

	return sb.String()
}

func systemPromptForRole(role string) string {
	switch role {
	case "story_analyst":
		return "你是一个专业的故事分析师。分析输入文本，提取主题、核心冲突和故事主线。输出严格 JSON 格式。"
	case "outline_planner":
		return "你是一个大纲规划师。将故事拆分为四个情节点（开端/发展/转折/高潮），每个包含标题、摘要和视觉目标。输出严格 JSON 格式。"
	case "character_analyst":
		return "你是一个角色分析师。从故事中提取主要人物，包括名称、描述、关系和动机。输出严格 JSON 格式。"
	case "scene_analyst":
		return "你是一个场景分析师。识别故事中的关键场景，包括名称、氛围和视觉元素。输出严格 JSON 格式。"
	case "prop_analyst":
		return "你是一个道具分析师。识别故事中的关键道具和线索物，包括名称、用途和场景关联。输出严格 JSON 格式。"
	case "screenwriter":
		return "你是一个编剧。将故事大纲和角色/场景/道具信息转化为剧集脚本，包含场景分解、对白和旁白。输出严格 JSON 格式。"
	case "director":
		return "你是一个导演。规划视觉连续性、镜头路线和关键帧，确保角色、场景和道具在视觉上保持一致。输出严格 JSON 格式。"
	case "cinematographer":
		return "你是一个摄影指导。为每个镜头规划镜头语言（景别、机位、运镜、构图、灯光），优化视觉叙事。输出严格 JSON 格式。"
	case "voice_subtitle":
		return "你是一个配音导演。为每个场景生成 TTS 脚本、字幕片段和配音风格建议。输出严格 JSON 格式。"
	default:
		return "你是一个 AI 助手。"
	}
}

func storyAnalystUserPrompt(sourceText string) string {
	return fmt.Sprintf(`分析以下故事文本：

%s

输出 JSON：
{
  "themes": ["主题1", "主题2"],
  "conflict": "核心冲突",
  "main_plot": "故事主线（50字内）"
}`, sourceText)
}

func outlinePlannerUserPrompt(sourceText string, prev any) string {
	context := ""
	if r, ok := prev.(*AgentResult); ok {
		context = "\n\n故事分析结果：\n" + r.Output
	}
	return fmt.Sprintf(`将以下故事拆分为四个情节点：

%s%s

输出 JSON：
{
  "beats": [
    {"code": "B01", "title": "开端", "summary": "...", "visual_goal": "..."},
    {"code": "B02", "title": "发展", "summary": "...", "visual_goal": "..."},
    {"code": "B03", "title": "转折", "summary": "...", "visual_goal": "..."},
    {"code": "B04", "title": "高潮", "summary": "...", "visual_goal": "..."}
  ]
}`, sourceText, context)
}

func characterAnalystUserPrompt(sourceText string, prev any) string {
	context := ""
	if r, ok := prev.(*AgentResult); ok {
		context = "\n\n大纲：\n" + r.Output
	}
	return fmt.Sprintf(`从以下故事中提取角色信息：

%s%s

输出 JSON：
{
  "characters": [
    {"code": "C01", "name": "名称", "description": "描述"}
  ]
}`, sourceText, context)
}

func sceneAnalystUserPrompt(sourceText string, prev any) string {
	context := ""
	if r, ok := prev.(*AgentResult); ok {
		context = "\n\n大纲：\n" + r.Output
	}
	return fmt.Sprintf(`从以下故事中识别场景：

%s%s

输出 JSON：
{
  "scenes": [
    {"code": "S01", "name": "名称", "description": "描述"}
  ]
}`, sourceText, context)
}

func propAnalystUserPrompt(sourceText string, prev any) string {
	context := ""
	if r, ok := prev.(*AgentResult); ok {
		context = "\n\n大纲：\n" + r.Output
	}
	return fmt.Sprintf(`从以下故事中识别关键道具：

%s%s

输出 JSON：
{
  "props": [
    {"code": "P01", "name": "名称", "description": "描述"}
  ]
}`, sourceText, context)
}

func screenwriterUserPrompt(sourceText string, bb *workflow.Blackboard) string {
	var context strings.Builder
	if r, ok := bb.Read("outline_planner"); ok {
		if ar, ok := r.(*AgentResult); ok {
			context.WriteString("\n\n大纲：\n" + ar.Output)
		}
	}
	for _, role := range []string{"character_analyst", "scene_analyst", "prop_analyst"} {
		if r, ok := bb.Read(role); ok {
			if ar, ok := r.(*AgentResult); ok {
				context.WriteString("\n\n" + role + " 产出：\n" + ar.Output)
			}
		}
	}
	return fmt.Sprintf(`将以下故事转化为剧集脚本，包含场景分解、角色对白和旁白：

%s%s

输出 JSON：
{
  "scenes": [
    {
      "code": "SC01",
      "title": "场景标题",
      "setting": "场景描述",
      "dialogues": [
        {"character": "角色名", "line": "台词", "direction": "表演指导"}
      ],
      "narration": "旁白文字"
    }
  ]
}`, sourceText, context.String())
}

func directorUserPrompt(sourceText string, prev any) string {
	context := ""
	if r, ok := prev.(*AgentResult); ok {
		context = "\n\n编剧脚本：\n" + r.Output
	}
	return fmt.Sprintf(`为以下剧集规划视觉连续性和镜头路线：

%s%s

输出 JSON：
{
  "visual_plan": {
    "continuity_notes": ["一致性要点1", "一致性要点2"],
    "key_frames": [
      {"scene": "SC01", "shot": "关键帧描述", "mood": "情绪基调"}
    ],
    "transition_notes": "场景转场建议"
  }
}`, sourceText, context)
}

func cinematographerUserPrompt(sourceText string, prev any) string {
	context := ""
	if r, ok := prev.(*AgentResult); ok {
		context = "\n\n编剧脚本：\n" + r.Output
	}
	return fmt.Sprintf(`为以下剧集的每个场景规划镜头语言：

%s%s

输出 JSON：
{
  "shots": [
    {
      "scene": "SC01",
      "shot_size": "MCU",
      "camera_angle": "eye-level",
      "camera_movement": "push-in",
      "composition": "rule-of-thirds",
      "lighting": "侧光 暖色调",
      "note": "镜头备注"
    }
  ]
}`, sourceText, context)
}

func voiceSubtitleUserPrompt(sourceText string, prev any) string {
	context := ""
	if r, ok := prev.(*AgentResult); ok {
		context = "\n\n编剧脚本：\n" + r.Output
	}
	return fmt.Sprintf(`为以下剧集生成配音脚本和字幕片段：

%s%s

输出 JSON：
{
  "voice_segments": [
    {
      "scene": "SC01",
      "character": "角色名",
      "text": "配音文字",
      "style": "配音风格（如：低沉/激昂/温柔）",
      "subtitle": "字幕文字",
      "duration_hint_ms": 3000
    }
  ]
}`, sourceText, context)
}

func extractHighlights(role string, content string) []string {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var parsed map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return []string{truncateStr(content, 80)}
	}

	switch role {
	case "story_analyst":
		var data struct {
			Themes []string `json:"themes"`
		}
		json.Unmarshal([]byte(content), &data)
		return data.Themes
	case "outline_planner":
		var data struct {
			Beats []struct {
				Code  string `json:"code"`
				Title string `json:"title"`
			} `json:"beats"`
		}
		json.Unmarshal([]byte(content), &data)
		titles := make([]string, 0, len(data.Beats))
		for _, b := range data.Beats {
			titles = append(titles, b.Code+" "+b.Title)
		}
		return titles
	case "character_analyst":
		return extractNames(content, "characters")
	case "scene_analyst":
		return extractNames(content, "scenes")
	case "prop_analyst":
		return extractNames(content, "props")
	case "screenwriter":
		return extractSceneHighlights(content)
	case "director":
		return extractDirectorHighlights(content)
	case "cinematographer":
		return extractCinematographerHighlights(content)
	case "voice_subtitle":
		return extractVoiceHighlights(content)
	}
	return nil
}

func extractNames(content string, key string) []string {
	var data map[string][]struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil
	}
	items := data[key]
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Code+" "+item.Name)
	}
	return names
}

func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

func extractSceneHighlights(content string) []string {
	var data struct {
		Scenes []struct {
			Code  string `json:"code"`
			Title string `json:"title"`
		} `json:"scenes"`
	}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil
	}
	highlights := make([]string, 0, len(data.Scenes))
	for _, s := range data.Scenes {
		highlights = append(highlights, s.Code+" "+s.Title)
	}
	return highlights
}

func extractDirectorHighlights(content string) []string {
	var data struct {
		VisualPlan struct {
			ContinuityNotes []string `json:"continuity_notes"`
		} `json:"visual_plan"`
	}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil
	}
	return data.VisualPlan.ContinuityNotes
}

func extractCinematographerHighlights(content string) []string {
	var data struct {
		Shots []struct {
			Scene    string `json:"scene"`
			ShotSize string `json:"shot_size"`
		} `json:"shots"`
	}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil
	}
	highlights := make([]string, 0, len(data.Shots))
	for _, s := range data.Shots {
		highlights = append(highlights, s.Scene+" "+s.ShotSize)
	}
	return highlights
}

func extractVoiceHighlights(content string) []string {
	var data struct {
		VoiceSegments []struct {
			Character string `json:"character"`
			Style     string `json:"style"`
		} `json:"voice_segments"`
	}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil
	}
	highlights := make([]string, 0, len(data.VoiceSegments))
	for _, v := range data.VoiceSegments {
		highlights = append(highlights, v.Character+" · "+v.Style)
	}
	return highlights
}
