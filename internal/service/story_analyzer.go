package service

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yibaiba/dramora/internal/domain"
)

var storySourceSplitter = regexp.MustCompile(`[。！？!?；;\n]+`)

type deterministicStoryAnalysis struct {
	summary        string
	themes         []string
	characterSeeds []string
	sceneSeeds     []string
	propSeeds      []string
	outline        []domain.StoryBeat
	agentOutputs   []domain.StoryAgentOutput
}

func analyzeStorySource(source domain.StorySource) deterministicStoryAnalysis {
	sentences := storySentences(source.ContentText)
	characters := storyItems("C", []string{"主角", "反派"}, storyCharacterNames(source.ContentText, source.Title))
	scenes := storyItems("S", []string{"开场场景", "冲突场景", "转折场景"}, storySceneNames(sentences))
	props := storyItems("P", []string{"关键道具", "线索物"}, storyPropNames(source.ContentText))
	outline := storyOutline(sentences)
	return deterministicStoryAnalysis{
		summary:        storySummary(source, sentences),
		themes:         storyThemes(source.ContentText),
		characterSeeds: characters,
		sceneSeeds:     scenes,
		propSeeds:      props,
		outline:        outline,
		agentOutputs:   storyAgentOutputs(characters, scenes, props, outline),
	}
}

func storySentences(content string) []string {
	parts := storySourceSplitter.Split(content, -1)
	sentences := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			sentences = append(sentences, value)
		}
	}
	return sentences
}

func storyItems(prefix string, defaults []string, candidates []string) []string {
	values := append([]string{}, candidates...)
	for len(values) < len(defaults) {
		values = append(values, defaults[len(values)])
	}
	items := make([]string, 0, len(values))
	for index, value := range values {
		items = append(items, fmt.Sprintf("%s%02d %s", prefix, index+1, value))
	}
	return items
}

func storyCharacterNames(content string, title string) []string {
	candidates := keywordsFromText(content, 4)
	if len(candidates) > 0 {
		return candidates[:minInt(len(candidates), 4)]
	}
	if strings.TrimSpace(title) != "" {
		return []string{strings.TrimSpace(title) + "主角"}
	}
	return nil
}

func storySceneNames(sentences []string) []string {
	names := make([]string, 0, 4)
	for index, sentence := range sentences {
		if index >= 4 {
			break
		}
		names = append(names, compactText(sentence, 12))
	}
	return names
}

func storyPropNames(content string) []string {
	keywords := []string{"剑", "令牌", "玉佩", "卷轴", "钥匙", "信", "面具", "法器"}
	props := make([]string, 0, 3)
	for _, keyword := range keywords {
		if strings.Contains(content, keyword) {
			props = append(props, keyword)
		}
	}
	return props
}

func storyOutline(sentences []string) []domain.StoryBeat {
	beats := make([]domain.StoryBeat, 0, 4)
	for index, title := range []string{"开端", "发展", "转折", "高潮"} {
		summary := sentenceAt(sentences, index)
		beats = append(beats, domain.StoryBeat{
			Code: fmt.Sprintf("B%02d", index+1), Title: title,
			Summary: summary, VisualGoal: visualGoal(title, summary),
		})
	}
	return beats
}

func storySummary(source domain.StorySource, sentences []string) string {
	lead := sentenceAt(sentences, 0)
	if strings.TrimSpace(source.Title) == "" {
		return "多 Agent 本地分析已提取故事主线：" + lead
	}
	return fmt.Sprintf("《%s》多 Agent 本地分析：%s", source.Title, lead)
}

func storyThemes(content string) []string {
	themes := []string{"成长", "抉择", "视觉对比"}
	if strings.Contains(content, "复仇") {
		themes[1] = "复仇"
	}
	if strings.Contains(content, "爱情") || strings.Contains(content, "守护") {
		themes = append(themes, "情感羁绊")
	}
	return themes
}

func storyAgentOutputs(
	characters []string,
	scenes []string,
	props []string,
	outline []domain.StoryBeat,
) []domain.StoryAgentOutput {
	return []domain.StoryAgentOutput{
		{Role: "story_analyst", Status: "succeeded", Output: "提炼主题、冲突和故事主线。", Highlights: []string{outline[0].Summary}},
		{Role: "outline_planner", Status: "succeeded", Output: "拆分四段式剧情大纲。", Highlights: beatTitles(outline)},
		{Role: "character_analyst", Status: "succeeded", Output: "抽取主要人物与关系线索。", Highlights: characters},
		{Role: "scene_analyst", Status: "succeeded", Output: "抽取可视化场景候选。", Highlights: scenes},
		{Role: "prop_analyst", Status: "succeeded", Output: "抽取关键道具和线索物。", Highlights: props},
	}
}

func keywordsFromText(content string, limit int) []string {
	fields := strings.FieldsFunc(content, func(r rune) bool {
		return strings.ContainsRune(" ，。！？；：、“”‘’（）()[]【】\n\t", r)
	})
	values := make([]string, 0, limit)
	seen := map[string]bool{}
	for _, field := range fields {
		if isKeywordCandidate(field) && !seen[field] {
			values = append(values, field)
			seen[field] = true
		}
		if len(values) >= limit {
			break
		}
	}
	return values
}

func isKeywordCandidate(value string) bool {
	size := len([]rune(value))
	return size >= 2 && size <= 6
}

func sentenceAt(sentences []string, index int) string {
	if index < len(sentences) {
		return compactText(sentences[index], 48)
	}
	return "围绕主角目标推进剧情，形成清晰的视觉段落。"
}

func compactText(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "..."
}

func visualGoal(title string, summary string) string {
	return fmt.Sprintf("%s段落需要突出%s", title, compactText(summary, 18))
}

func beatTitles(outline []domain.StoryBeat) []string {
	titles := make([]string, 0, len(outline))
	for _, beat := range outline {
		titles = append(titles, beat.Code+" "+beat.Title)
	}
	return titles
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
