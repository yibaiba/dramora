package workflow

// Phase1Graph: 5 节点，story_analyst → outline_planner → character/scene/prop 并行
var Phase1Graph = &Graph{
	Nodes: []Node{
		{ID: "story_analyst", Kind: NodeKindStoryAnalysis},
		{ID: "outline_planner", Kind: NodeKindStoryAnalysis},
		{ID: "character_analyst", Kind: NodeKindCharacterDesign},
		{ID: "scene_analyst", Kind: NodeKindSceneDesign},
		{ID: "prop_analyst", Kind: NodeKindPropDesign},
	},
	Edges: []Edge{
		{FromNodeID: "story_analyst", ToNodeID: "outline_planner"},
		{FromNodeID: "outline_planner", ToNodeID: "character_analyst"},
		{FromNodeID: "outline_planner", ToNodeID: "scene_analyst"},
		{FromNodeID: "outline_planner", ToNodeID: "prop_analyst"},
	},
}

// Phase2Graph: 9 节点，在 Phase1 基础上增加 screenwriter → director/cinematographer/voice 并行
var Phase2Graph = &Graph{
	Nodes: []Node{
		{ID: "story_analyst", Kind: NodeKindStoryAnalysis},
		{ID: "outline_planner", Kind: NodeKindStoryAnalysis},
		{ID: "character_analyst", Kind: NodeKindCharacterDesign},
		{ID: "scene_analyst", Kind: NodeKindSceneDesign},
		{ID: "prop_analyst", Kind: NodeKindPropDesign},
		{ID: "screenwriter", Kind: NodeKindScreenwriter},
		{ID: "director", Kind: NodeKindDirector},
		{ID: "cinematographer", Kind: NodeKindCinematographer},
		{ID: "voice_subtitle", Kind: NodeKindVoiceSubtitle},
	},
	Edges: []Edge{
		// Phase 1 edges
		{FromNodeID: "story_analyst", ToNodeID: "outline_planner"},
		{FromNodeID: "outline_planner", ToNodeID: "character_analyst"},
		{FromNodeID: "outline_planner", ToNodeID: "scene_analyst"},
		{FromNodeID: "outline_planner", ToNodeID: "prop_analyst"},
		// Phase 2 edges: screenwriter depends on all Phase 1 outputs
		{FromNodeID: "character_analyst", ToNodeID: "screenwriter"},
		{FromNodeID: "scene_analyst", ToNodeID: "screenwriter"},
		{FromNodeID: "prop_analyst", ToNodeID: "screenwriter"},
		// director/cinematographer/voice depend on screenwriter
		{FromNodeID: "screenwriter", ToNodeID: "director"},
		{FromNodeID: "screenwriter", ToNodeID: "cinematographer"},
		{FromNodeID: "screenwriter", ToNodeID: "voice_subtitle"},
	},
}
