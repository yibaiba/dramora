package domain

import "time"

type WorkflowRun struct {
	ID        string
	ProjectID string
	EpisodeID string
	Status    WorkflowRunStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type GenerationJob struct {
	ID             string
	ProjectID      string
	EpisodeID      string
	WorkflowRunID  string
	Provider       string
	Model          string
	TaskType       string
	Status         GenerationJobStatus
	Prompt         string
	Params         map[string]any
	ProviderTaskID string
	ResultAssetID  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ApprovalGate struct {
	ID            string
	ProjectID     string
	EpisodeID     string
	WorkflowRunID string
	GateType      string
	SubjectType   string
	SubjectID     string
	Status        ApprovalGateStatus
	ReviewedBy    string
	ReviewNote    string
	ReviewedAt    time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type StoryAnalysisStatus string

const (
	StoryAnalysisStatusGenerated StoryAnalysisStatus = "generated"
	StoryAnalysisStatusApproved  StoryAnalysisStatus = "approved"
)

type StoryAnalysis struct {
	ID              string
	ProjectID       string
	EpisodeID       string
	StorySourceID   string
	WorkflowRunID   string
	GenerationJobID string
	Version         int
	Status          StoryAnalysisStatus
	Summary         string
	Themes          []string
	CharacterSeeds  []string
	SceneSeeds      []string
	PropSeeds       []string
	Outline         []StoryBeat
	AgentOutputs    []StoryAgentOutput
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type StorySource struct {
	ID          string
	ProjectID   string
	EpisodeID   string
	SourceType  string
	Title       string
	ContentText string
	Language    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type StoryBeat struct {
	Code       string `json:"code"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	VisualGoal string `json:"visual_goal"`
}

type StoryAgentOutput struct {
	Role       string   `json:"role"`
	Status     string   `json:"status"`
	Output     string   `json:"output"`
	Highlights []string `json:"highlights"`
}

type CharacterBiblePalette struct {
	Skin    string `json:"skin"`
	Hair    string `json:"hair"`
	Accent  string `json:"accent"`
	Eyes    string `json:"eyes"`
	Costume string `json:"costume"`
}

type CharacterBibleReferenceAsset struct {
	Angle   string `json:"angle"`
	AssetID string `json:"asset_id"`
}

type CharacterBible struct {
	Anchor          string                         `json:"anchor"`
	Palette         CharacterBiblePalette          `json:"palette"`
	Expressions     []string                       `json:"expressions"`
	ReferenceAngles []string                       `json:"reference_angles"`
	ReferenceAssets []CharacterBibleReferenceAsset `json:"reference_assets"`
	Wardrobe        string                         `json:"wardrobe"`
	Notes           string                         `json:"notes"`
}

type Asset struct {
	ID        string
	ProjectID string
	EpisodeID string
	Kind      string
	Purpose   string
	URI       string
	Status    AssetStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Timeline struct {
	ID         string
	EpisodeID  string
	Status     TimelineStatus
	Version    int
	DurationMS int
	Tracks     []TimelineTrack
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Character struct {
	ID              string
	ProjectID       string
	EpisodeID       string
	StoryAnalysisID string
	Code            string
	Name            string
	Description     string
	CharacterBible  *CharacterBible
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Scene struct {
	ID              string
	ProjectID       string
	EpisodeID       string
	StoryAnalysisID string
	Code            string
	Name            string
	Description     string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Prop struct {
	ID              string
	ProjectID       string
	EpisodeID       string
	StoryAnalysisID string
	Code            string
	Name            string
	Description     string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type StoryboardShot struct {
	ID              string
	ProjectID       string
	EpisodeID       string
	StoryAnalysisID string
	SceneID         string
	Code            string
	Title           string
	Description     string
	Prompt          string
	Position        int
	DurationMS      int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type PromptTimeSlice struct {
	StartMS     int
	EndMS       int
	Prompt      string
	CameraWork  string
	ShotSize    string
	VisualFocus string
}

type PromptReferenceBinding struct {
	Token   string
	Role    string
	AssetID string
	Kind    string
	URI     string
}

type ShotPromptPack struct {
	ID                string
	ProjectID         string
	EpisodeID         string
	ShotID            string
	Provider          string
	Model             string
	Preset            string
	TaskType          string
	DirectPrompt      string
	NegativePrompt    string
	TimeSlices        []PromptTimeSlice
	ReferenceBindings []PromptReferenceBinding
	Params            map[string]any
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type TimelineTrack struct {
	ID         string
	TimelineID string
	Kind       string
	Name       string
	Position   int
	Clips      []TimelineClip
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type TimelineClip struct {
	ID          string
	TimelineID  string
	TrackID     string
	AssetID     string
	Kind        string
	StartMS     int
	DurationMS  int
	TrimStartMS int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Export struct {
	ID         string
	TimelineID string
	Status     ExportStatus
	Format     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
