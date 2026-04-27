package httpapi

import (
	"time"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

type workflowRunResponse struct {
	ID        string                   `json:"id"`
	ProjectID string                   `json:"project_id"`
	EpisodeID string                   `json:"episode_id"`
	Status    domain.WorkflowRunStatus `json:"status"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
}

type generationJobResponse struct {
	ID            string                     `json:"id"`
	ProjectID     string                     `json:"project_id"`
	EpisodeID     string                     `json:"episode_id"`
	WorkflowRunID string                     `json:"workflow_run_id"`
	Provider      string                     `json:"provider"`
	Model         string                     `json:"model"`
	TaskType      string                     `json:"task_type"`
	Status        domain.GenerationJobStatus `json:"status"`
	CreatedAt     time.Time                  `json:"created_at"`
	UpdatedAt     time.Time                  `json:"updated_at"`
}

type storyAnalysisResponse struct {
	ID              string                     `json:"id"`
	ProjectID       string                     `json:"project_id"`
	EpisodeID       string                     `json:"episode_id"`
	WorkflowRunID   string                     `json:"workflow_run_id"`
	GenerationJobID string                     `json:"generation_job_id"`
	Version         int                        `json:"version"`
	Status          domain.StoryAnalysisStatus `json:"status"`
	Summary         string                     `json:"summary"`
	Themes          []string                   `json:"themes"`
	CharacterSeeds  []string                   `json:"character_seeds"`
	SceneSeeds      []string                   `json:"scene_seeds"`
	PropSeeds       []string                   `json:"prop_seeds"`
	CreatedAt       time.Time                  `json:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at"`
}

type timelineResponse struct {
	ID         string                  `json:"id"`
	EpisodeID  string                  `json:"episode_id"`
	Status     domain.TimelineStatus   `json:"status"`
	Version    int                     `json:"version"`
	DurationMS int                     `json:"duration_ms"`
	Tracks     []timelineTrackResponse `json:"tracks"`
	CreatedAt  time.Time               `json:"created_at"`
	UpdatedAt  time.Time               `json:"updated_at"`
}

type storyMapResponse struct {
	Characters []characterResponse `json:"characters"`
	Scenes     []sceneResponse     `json:"scenes"`
	Props      []propResponse      `json:"props"`
}

type characterResponse struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	EpisodeID   string    `json:"episode_id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type sceneResponse characterResponse
type propResponse characterResponse

type storyboardShotResponse struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	EpisodeID   string    `json:"episode_id"`
	SceneID     string    `json:"scene_id"`
	Code        string    `json:"code"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Prompt      string    `json:"prompt"`
	Position    int       `json:"position"`
	DurationMS  int       `json:"duration_ms"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type promptTimeSliceResponse struct {
	StartMS     int    `json:"start_ms"`
	EndMS       int    `json:"end_ms"`
	Prompt      string `json:"prompt"`
	CameraWork  string `json:"camera_work"`
	ShotSize    string `json:"shot_size"`
	VisualFocus string `json:"visual_focus"`
}

type promptReferenceBindingResponse struct {
	Token   string `json:"token"`
	Role    string `json:"role"`
	AssetID string `json:"asset_id"`
	Kind    string `json:"kind"`
	URI     string `json:"uri"`
}

type shotPromptPackResponse struct {
	ID                string                           `json:"id"`
	ProjectID         string                           `json:"project_id"`
	EpisodeID         string                           `json:"episode_id"`
	ShotID            string                           `json:"shot_id"`
	Provider          string                           `json:"provider"`
	Model             string                           `json:"model"`
	Preset            string                           `json:"preset"`
	TaskType          string                           `json:"task_type"`
	DirectPrompt      string                           `json:"direct_prompt"`
	NegativePrompt    string                           `json:"negative_prompt"`
	TimeSlices        []promptTimeSliceResponse        `json:"time_slices"`
	ReferenceBindings []promptReferenceBindingResponse `json:"reference_bindings"`
	Params            map[string]any                   `json:"params"`
	CreatedAt         time.Time                        `json:"created_at"`
	UpdatedAt         time.Time                        `json:"updated_at"`
}

type assetResponse struct {
	ID        string             `json:"id"`
	ProjectID string             `json:"project_id"`
	EpisodeID string             `json:"episode_id"`
	Kind      string             `json:"kind"`
	Purpose   string             `json:"purpose"`
	URI       string             `json:"uri"`
	Status    domain.AssetStatus `json:"status"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

type timelineTrackResponse struct {
	ID        string                 `json:"id"`
	Kind      string                 `json:"kind"`
	Name      string                 `json:"name"`
	Position  int                    `json:"position"`
	Clips     []timelineClipResponse `json:"clips"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type timelineClipResponse struct {
	ID          string    `json:"id"`
	AssetID     string    `json:"asset_id"`
	Kind        string    `json:"kind"`
	StartMS     int       `json:"start_ms"`
	DurationMS  int       `json:"duration_ms"`
	TrimStartMS int       `json:"trim_start_ms"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type exportResponse struct {
	ID         string              `json:"id"`
	TimelineID string              `json:"timeline_id"`
	Status     domain.ExportStatus `json:"status"`
	Format     string              `json:"format"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
}

func workflowRunDTO(run domain.WorkflowRun) workflowRunResponse {
	return workflowRunResponse{
		ID:        run.ID,
		ProjectID: run.ProjectID,
		EpisodeID: run.EpisodeID,
		Status:    run.Status,
		CreatedAt: run.CreatedAt,
		UpdatedAt: run.UpdatedAt,
	}
}

func generationJobDTO(job domain.GenerationJob) generationJobResponse {
	return generationJobResponse{
		ID:            job.ID,
		ProjectID:     job.ProjectID,
		EpisodeID:     job.EpisodeID,
		WorkflowRunID: job.WorkflowRunID,
		Provider:      job.Provider,
		Model:         job.Model,
		TaskType:      job.TaskType,
		Status:        job.Status,
		CreatedAt:     job.CreatedAt,
		UpdatedAt:     job.UpdatedAt,
	}
}

func storyAnalysisDTO(analysis domain.StoryAnalysis) storyAnalysisResponse {
	return storyAnalysisResponse{
		ID:              analysis.ID,
		ProjectID:       analysis.ProjectID,
		EpisodeID:       analysis.EpisodeID,
		WorkflowRunID:   analysis.WorkflowRunID,
		GenerationJobID: analysis.GenerationJobID,
		Version:         analysis.Version,
		Status:          analysis.Status,
		Summary:         analysis.Summary,
		Themes:          analysis.Themes,
		CharacterSeeds:  analysis.CharacterSeeds,
		SceneSeeds:      analysis.SceneSeeds,
		PropSeeds:       analysis.PropSeeds,
		CreatedAt:       analysis.CreatedAt,
		UpdatedAt:       analysis.UpdatedAt,
	}
}

func timelineDTO(timeline domain.Timeline) timelineResponse {
	return timelineResponse{
		ID:         timeline.ID,
		EpisodeID:  timeline.EpisodeID,
		Status:     timeline.Status,
		Version:    timeline.Version,
		DurationMS: timeline.DurationMS,
		Tracks:     timelineTrackDTOs(timeline.Tracks),
		CreatedAt:  timeline.CreatedAt,
		UpdatedAt:  timeline.UpdatedAt,
	}
}

func storyMapDTO(storyMap repo.StoryMap) storyMapResponse {
	return storyMapResponse{
		Characters: characterDTOs(storyMap.Characters),
		Scenes:     sceneDTOs(storyMap.Scenes),
		Props:      propDTOs(storyMap.Props),
	}
}

func characterDTOs(items []domain.Character) []characterResponse {
	responses := make([]characterResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, characterResponse{
			ID: item.ID, ProjectID: item.ProjectID, EpisodeID: item.EpisodeID,
			Code: item.Code, Name: item.Name, Description: item.Description,
			CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}
	return responses
}

func sceneDTOs(items []domain.Scene) []sceneResponse {
	responses := make([]sceneResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, sceneResponse(characterResponse{
			ID: item.ID, ProjectID: item.ProjectID, EpisodeID: item.EpisodeID,
			Code: item.Code, Name: item.Name, Description: item.Description,
			CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		}))
	}
	return responses
}

func propDTOs(items []domain.Prop) []propResponse {
	responses := make([]propResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, propResponse(characterResponse{
			ID: item.ID, ProjectID: item.ProjectID, EpisodeID: item.EpisodeID,
			Code: item.Code, Name: item.Name, Description: item.Description,
			CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		}))
	}
	return responses
}

func storyboardShotDTOs(items []domain.StoryboardShot) []storyboardShotResponse {
	responses := make([]storyboardShotResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, storyboardShotResponse{
			ID: item.ID, ProjectID: item.ProjectID, EpisodeID: item.EpisodeID,
			SceneID: item.SceneID, Code: item.Code, Title: item.Title,
			Description: item.Description, Prompt: item.Prompt,
			Position: item.Position, DurationMS: item.DurationMS,
			CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}
	return responses
}

func shotPromptPackDTO(item domain.ShotPromptPack) shotPromptPackResponse {
	return shotPromptPackResponse{
		ID: item.ID, ProjectID: item.ProjectID, EpisodeID: item.EpisodeID,
		ShotID: item.ShotID, Provider: item.Provider, Model: item.Model,
		Preset: item.Preset, TaskType: item.TaskType, DirectPrompt: item.DirectPrompt,
		NegativePrompt: item.NegativePrompt, TimeSlices: promptTimeSliceDTOs(item.TimeSlices),
		ReferenceBindings: promptReferenceBindingDTOs(item.ReferenceBindings), Params: item.Params,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func promptTimeSliceDTOs(items []domain.PromptTimeSlice) []promptTimeSliceResponse {
	responses := make([]promptTimeSliceResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, promptTimeSliceResponse{
			StartMS: item.StartMS, EndMS: item.EndMS, Prompt: item.Prompt,
			CameraWork: item.CameraWork, ShotSize: item.ShotSize, VisualFocus: item.VisualFocus,
		})
	}
	return responses
}

func promptReferenceBindingDTOs(items []domain.PromptReferenceBinding) []promptReferenceBindingResponse {
	responses := make([]promptReferenceBindingResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, promptReferenceBindingResponse{
			Token: item.Token, Role: item.Role, AssetID: item.AssetID,
			Kind: item.Kind, URI: item.URI,
		})
	}
	return responses
}

func assetDTOs(items []domain.Asset) []assetResponse {
	responses := make([]assetResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, assetDTO(item))
	}
	return responses
}

func assetDTO(item domain.Asset) assetResponse {
	return assetResponse{
		ID: item.ID, ProjectID: item.ProjectID, EpisodeID: item.EpisodeID,
		Kind: item.Kind, Purpose: item.Purpose, URI: item.URI, Status: item.Status,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func timelineTrackDTOs(items []domain.TimelineTrack) []timelineTrackResponse {
	responses := make([]timelineTrackResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, timelineTrackResponse{
			ID: item.ID, Kind: item.Kind, Name: item.Name, Position: item.Position,
			Clips: timelineClipDTOs(item.Clips), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}
	return responses
}

func timelineClipDTOs(items []domain.TimelineClip) []timelineClipResponse {
	responses := make([]timelineClipResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, timelineClipResponse{
			ID: item.ID, AssetID: item.AssetID, Kind: item.Kind, StartMS: item.StartMS,
			DurationMS: item.DurationMS, TrimStartMS: item.TrimStartMS,
			CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}
	return responses
}

func exportDTO(item domain.Export) exportResponse {
	return exportResponse{
		ID: item.ID, TimelineID: item.TimelineID, Status: item.Status,
		Format: item.Format, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}
