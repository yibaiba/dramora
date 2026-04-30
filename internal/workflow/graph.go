package workflow

type NodeKind string

const (
	NodeKindStoryAnalysis     NodeKind = "story_analysis"
	NodeKindCharacterDesign   NodeKind = "character_design"
	NodeKindSceneDesign       NodeKind = "scene_design"
	NodeKindPropDesign        NodeKind = "prop_design"
	NodeKindScreenwriter      NodeKind = "screenwriter"
	NodeKindDirector          NodeKind = "director"
	NodeKindCinematographer   NodeKind = "cinematographer"
	NodeKindVoiceSubtitle     NodeKind = "voice_subtitle"
	NodeKindStoryboard        NodeKind = "storyboard"
	NodeKindPromptEngineering NodeKind = "prompt_engineering"
	NodeKindVideoGeneration   NodeKind = "video_generation"
	NodeKindContinuityReview  NodeKind = "continuity_review"
	NodeKindTimelineExport    NodeKind = "timeline_export"
)

type Node struct {
	ID   string
	Kind NodeKind
}

type Edge struct {
	FromNodeID string
	ToNodeID   string
}

type Graph struct {
	Nodes []Node
	Edges []Edge
}
