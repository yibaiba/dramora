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
	ID            string
	ProjectID     string
	EpisodeID     string
	WorkflowRunID string
	Provider      string
	Model         string
	TaskType      string
	Status        GenerationJobStatus
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
	WorkflowRunID   string
	GenerationJobID string
	Version         int
	Status          StoryAnalysisStatus
	Summary         string
	Themes          []string
	CharacterSeeds  []string
	SceneSeeds      []string
	PropSeeds       []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
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
