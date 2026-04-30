package httpapi

import (
	"time"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
	"github.com/yibaiba/dramora/internal/service"
)

type workflowRunResponse struct {
	ID                string                             `json:"id"`
	ProjectID         string                             `json:"project_id"`
	EpisodeID         string                             `json:"episode_id"`
	Status            domain.WorkflowRunStatus           `json:"status"`
	CheckpointSummary *workflowCheckpointSummaryResponse `json:"checkpoint_summary,omitempty"`
	NodeRuns          []workflowNodeRunResponse          `json:"node_runs,omitempty"`
	CreatedAt         time.Time                          `json:"created_at"`
	UpdatedAt         time.Time                          `json:"updated_at"`
}

type workflowCheckpointSummaryResponse struct {
	Sequence        uint64    `json:"sequence"`
	SavedAt         time.Time `json:"saved_at"`
	CompletedNodes  int       `json:"completed_nodes"`
	WaitingNodes    int       `json:"waiting_nodes"`
	RunningNodes    int       `json:"running_nodes"`
	FailedNodes     int       `json:"failed_nodes"`
	SkippedNodes    int       `json:"skipped_nodes"`
	BlackboardRoles []string  `json:"blackboard_roles"`
}

type workflowNodeRunResponse struct {
	NodeID          string                       `json:"node_id"`
	Kind            string                       `json:"kind"`
	Status          domain.WorkflowNodeRunStatus `json:"status"`
	Summary         string                       `json:"summary"`
	Highlights      []string                     `json:"highlights"`
	ErrorMessage    string                       `json:"error_message"`
	UpstreamNodeIDs []string                     `json:"upstream_node_ids"`
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
	ResultAssetID string                     `json:"result_asset_id"`
	CreatedAt     time.Time                  `json:"created_at"`
	UpdatedAt     time.Time                  `json:"updated_at"`
}

type approvalGateResponse struct {
	ID            string                    `json:"id"`
	ProjectID     string                    `json:"project_id"`
	EpisodeID     string                    `json:"episode_id"`
	WorkflowRunID string                    `json:"workflow_run_id"`
	GateType      string                    `json:"gate_type"`
	SubjectType   string                    `json:"subject_type"`
	SubjectID     string                    `json:"subject_id"`
	Status        domain.ApprovalGateStatus `json:"status"`
	ReviewedBy    string                    `json:"reviewed_by"`
	ReviewNote    string                    `json:"review_note"`
	ReviewedAt    time.Time                 `json:"reviewed_at"`
	CreatedAt     time.Time                 `json:"created_at"`
	UpdatedAt     time.Time                 `json:"updated_at"`
}

type storySourceResponse struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	EpisodeID   string    `json:"episode_id"`
	SourceType  string    `json:"source_type"`
	Title       string    `json:"title"`
	ContentText string    `json:"content_text"`
	Language    string    `json:"language"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type storyAnalysisResponse struct {
	ID              string                     `json:"id"`
	ProjectID       string                     `json:"project_id"`
	EpisodeID       string                     `json:"episode_id"`
	StorySourceID   string                     `json:"story_source_id"`
	WorkflowRunID   string                     `json:"workflow_run_id"`
	GenerationJobID string                     `json:"generation_job_id"`
	Version         int                        `json:"version"`
	Status          domain.StoryAnalysisStatus `json:"status"`
	Summary         string                     `json:"summary"`
	Themes          []string                   `json:"themes"`
	CharacterSeeds  []string                   `json:"character_seeds"`
	SceneSeeds      []string                   `json:"scene_seeds"`
	PropSeeds       []string                   `json:"prop_seeds"`
	Outline         []domain.StoryBeat         `json:"outline"`
	AgentOutputs    []domain.StoryAgentOutput  `json:"agent_outputs"`
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

type characterBiblePaletteResponse struct {
	Skin    string `json:"skin"`
	Hair    string `json:"hair"`
	Accent  string `json:"accent"`
	Eyes    string `json:"eyes"`
	Costume string `json:"costume"`
}

type characterBibleResponse struct {
	Anchor          string                                 `json:"anchor"`
	Palette         characterBiblePaletteResponse          `json:"palette"`
	Expressions     []string                               `json:"expressions"`
	ReferenceAngles []string                               `json:"reference_angles"`
	ReferenceAssets []characterBibleReferenceAssetResponse `json:"reference_assets"`
	Wardrobe        string                                 `json:"wardrobe"`
	Notes           string                                 `json:"notes"`
}

type characterBibleReferenceAssetResponse struct {
	Angle   string `json:"angle"`
	AssetID string `json:"asset_id"`
}

type characterResponse struct {
	ID             string                  `json:"id"`
	ProjectID      string                  `json:"project_id"`
	EpisodeID      string                  `json:"episode_id"`
	Code           string                  `json:"code"`
	Name           string                  `json:"name"`
	Description    string                  `json:"description"`
	CharacterBible *characterBibleResponse `json:"character_bible,omitempty"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
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

type storyboardShotPromptPackSummaryResponse struct {
	ID        string    `json:"id"`
	ShotID    string    `json:"shot_id"`
	Provider  string    `json:"provider"`
	Model     string    `json:"model"`
	Preset    string    `json:"preset"`
	TaskType  string    `json:"task_type"`
	UpdatedAt time.Time `json:"updated_at"`
}

type storyboardWorkspaceShotResponse struct {
	ID                  string                                   `json:"id"`
	ProjectID           string                                   `json:"project_id"`
	EpisodeID           string                                   `json:"episode_id"`
	SceneID             string                                   `json:"scene_id"`
	Code                string                                   `json:"code"`
	Title               string                                   `json:"title"`
	Description         string                                   `json:"description"`
	Prompt              string                                   `json:"prompt"`
	Position            int                                      `json:"position"`
	DurationMS          int                                      `json:"duration_ms"`
	Scene               *sceneResponse                           `json:"scene"`
	PromptPack          *storyboardShotPromptPackSummaryResponse `json:"prompt_pack"`
	LatestGenerationJob *generationJobResponse                   `json:"latest_generation_job"`
	CreatedAt           time.Time                                `json:"created_at"`
	UpdatedAt           time.Time                                `json:"updated_at"`
}

type storyboardWorkspaceSummaryResponse struct {
	AnalysisCount             int  `json:"analysis_count"`
	StoryMapReady             bool `json:"story_map_ready"`
	ReadyAssetsCount          int  `json:"ready_assets_count"`
	PendingApprovalGatesCount int  `json:"pending_approval_gates_count"`
}

type storyboardWorkspaceResponse struct {
	EpisodeID       string                             `json:"episode_id"`
	Summary         storyboardWorkspaceSummaryResponse `json:"summary"`
	StoryMap        storyMapResponse                   `json:"story_map"`
	StoryboardShots []storyboardWorkspaceShotResponse  `json:"storyboard_shots"`
	Assets          []assetResponse                    `json:"assets"`
	ApprovalGates   []approvalGateResponse             `json:"approval_gates"`
	GenerationJobs  []generationJobResponse            `json:"generation_jobs"`
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

func workflowRunDTO(
	run domain.WorkflowRun,
	checkpoint *service.WorkflowCheckpointSummary,
	nodeRuns []service.WorkflowNodeDetail,
) workflowRunResponse {
	var checkpointResponse *workflowCheckpointSummaryResponse
	if checkpoint != nil {
		checkpointResponse = &workflowCheckpointSummaryResponse{
			Sequence:        checkpoint.Sequence,
			SavedAt:         checkpoint.SavedAt,
			CompletedNodes:  checkpoint.CompletedNodes,
			WaitingNodes:    checkpoint.WaitingNodes,
			RunningNodes:    checkpoint.RunningNodes,
			FailedNodes:     checkpoint.FailedNodes,
			SkippedNodes:    checkpoint.SkippedNodes,
			BlackboardRoles: append([]string(nil), checkpoint.BlackboardRoles...),
		}
	}
	nodeRunResponses := make([]workflowNodeRunResponse, 0, len(nodeRuns))
	for _, nodeRun := range nodeRuns {
		nodeRunResponses = append(nodeRunResponses, workflowNodeRunResponse{
			NodeID:          nodeRun.NodeID,
			Kind:            string(nodeRun.Kind),
			Status:          nodeRun.Status,
			Summary:         nodeRun.Summary,
			Highlights:      append([]string(nil), nodeRun.Highlights...),
			ErrorMessage:    nodeRun.ErrorMessage,
			UpstreamNodeIDs: append([]string(nil), nodeRun.UpstreamNodeIDs...),
		})
	}
	return workflowRunResponse{
		ID:                run.ID,
		ProjectID:         run.ProjectID,
		EpisodeID:         run.EpisodeID,
		Status:            run.Status,
		CheckpointSummary: checkpointResponse,
		NodeRuns:          nodeRunResponses,
		CreatedAt:         run.CreatedAt,
		UpdatedAt:         run.UpdatedAt,
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
		ResultAssetID: job.ResultAssetID,
		CreatedAt:     job.CreatedAt,
		UpdatedAt:     job.UpdatedAt,
	}
}

func generationJobDTOs(jobs []domain.GenerationJob) []generationJobResponse {
	responses := make([]generationJobResponse, 0, len(jobs))
	for _, job := range jobs {
		responses = append(responses, generationJobDTO(job))
	}
	return responses
}

type generationJobEventResponse struct {
	ID        string                     `json:"id"`
	JobID     string                     `json:"generation_job_id"`
	Status    domain.GenerationJobStatus `json:"status"`
	Message   string                     `json:"message"`
	CreatedAt time.Time                  `json:"created_at"`
}

type generationJobRecoverySummaryResponse struct {
	IsTerminal       bool                       `json:"is_terminal"`
	IsRecoverable    bool                       `json:"is_recoverable"`
	CurrentStatus    domain.GenerationJobStatus `json:"current_status"`
	StatusEnteredAt  time.Time                  `json:"status_entered_at"`
	LastEventAt      time.Time                  `json:"last_event_at"`
	StatusEventCount int                        `json:"status_event_count"`
	TotalEventCount  int                        `json:"total_event_count"`
	NextHint         string                     `json:"next_hint"`
}

type generationJobRecoveryResponse struct {
	Job     generationJobResponse                `json:"generation_job"`
	Events  []generationJobEventResponse         `json:"events"`
	Summary generationJobRecoverySummaryResponse `json:"summary"`
}

func generationJobRecoveryDTO(recovery service.GenerationJobRecovery) generationJobRecoveryResponse {
	events := make([]generationJobEventResponse, 0, len(recovery.Events))
	for _, ev := range recovery.Events {
		events = append(events, generationJobEventResponse{
			ID:        ev.ID,
			JobID:     ev.GenerationJobID,
			Status:    ev.Status,
			Message:   ev.Message,
			CreatedAt: ev.CreatedAt,
		})
	}
	return generationJobRecoveryResponse{
		Job:    generationJobDTO(recovery.Job),
		Events: events,
		Summary: generationJobRecoverySummaryResponse{
			IsTerminal:       recovery.Summary.IsTerminal,
			IsRecoverable:    recovery.Summary.IsRecoverable,
			CurrentStatus:    recovery.Summary.CurrentStatus,
			StatusEnteredAt:  recovery.Summary.StatusEnteredAt,
			LastEventAt:      recovery.Summary.LastEventAt,
			StatusEventCount: recovery.Summary.StatusEventCount,
			TotalEventCount:  recovery.Summary.TotalEventCount,
			NextHint:         recovery.Summary.NextHint,
		},
	}
}

func approvalGateDTO(gate domain.ApprovalGate) approvalGateResponse {
	return approvalGateResponse{
		ID: gate.ID, ProjectID: gate.ProjectID, EpisodeID: gate.EpisodeID,
		WorkflowRunID: gate.WorkflowRunID, GateType: gate.GateType,
		SubjectType: gate.SubjectType, SubjectID: gate.SubjectID,
		Status: gate.Status, ReviewedBy: gate.ReviewedBy, ReviewNote: gate.ReviewNote,
		ReviewedAt: gate.ReviewedAt, CreatedAt: gate.CreatedAt, UpdatedAt: gate.UpdatedAt,
	}
}

func approvalGateDTOs(gates []domain.ApprovalGate) []approvalGateResponse {
	responses := make([]approvalGateResponse, 0, len(gates))
	for _, gate := range gates {
		responses = append(responses, approvalGateDTO(gate))
	}
	return responses
}

func storySourceDTO(source domain.StorySource) storySourceResponse {
	return storySourceResponse{
		ID: source.ID, ProjectID: source.ProjectID, EpisodeID: source.EpisodeID,
		SourceType: source.SourceType, Title: source.Title, ContentText: source.ContentText,
		Language: source.Language, CreatedAt: source.CreatedAt, UpdatedAt: source.UpdatedAt,
	}
}

func storyAnalysisDTO(analysis domain.StoryAnalysis) storyAnalysisResponse {
	return storyAnalysisResponse{
		ID:              analysis.ID,
		ProjectID:       analysis.ProjectID,
		EpisodeID:       analysis.EpisodeID,
		StorySourceID:   analysis.StorySourceID,
		WorkflowRunID:   analysis.WorkflowRunID,
		GenerationJobID: analysis.GenerationJobID,
		Version:         analysis.Version,
		Status:          analysis.Status,
		Summary:         analysis.Summary,
		Themes:          analysis.Themes,
		CharacterSeeds:  analysis.CharacterSeeds,
		SceneSeeds:      analysis.SceneSeeds,
		PropSeeds:       analysis.PropSeeds,
		Outline:         analysis.Outline,
		AgentOutputs:    analysis.AgentOutputs,
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
		responses = append(responses, characterDTO(item))
	}
	return responses
}

func characterDTO(item domain.Character) characterResponse {
	return characterResponse{
		ID: item.ID, ProjectID: item.ProjectID, EpisodeID: item.EpisodeID,
		Code: item.Code, Name: item.Name, Description: item.Description,
		CharacterBible: characterBibleDTO(item.CharacterBible),
		CreatedAt:      item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
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

func characterBibleDTO(bible *domain.CharacterBible) *characterBibleResponse {
	if bible == nil {
		return nil
	}
	return &characterBibleResponse{
		Anchor: bible.Anchor,
		Palette: characterBiblePaletteResponse{
			Skin:    bible.Palette.Skin,
			Hair:    bible.Palette.Hair,
			Accent:  bible.Palette.Accent,
			Eyes:    bible.Palette.Eyes,
			Costume: bible.Palette.Costume,
		},
		Expressions:     bible.Expressions,
		ReferenceAngles: bible.ReferenceAngles,
		ReferenceAssets: characterBibleReferenceAssetDTOs(bible.ReferenceAssets),
		Wardrobe:        bible.Wardrobe,
		Notes:           bible.Notes,
	}
}

func characterBibleReferenceAssetDTOs(items []domain.CharacterBibleReferenceAsset) []characterBibleReferenceAssetResponse {
	responses := make([]characterBibleReferenceAssetResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, characterBibleReferenceAssetResponse{
			Angle:   item.Angle,
			AssetID: item.AssetID,
		})
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

func storyboardWorkspaceDTO(item service.StoryboardWorkspace) storyboardWorkspaceResponse {
	return storyboardWorkspaceResponse{
		EpisodeID:       item.EpisodeID,
		Summary:         storyboardWorkspaceSummaryDTO(item.Summary),
		StoryMap:        storyMapDTO(item.StoryMap),
		StoryboardShots: storyboardWorkspaceShotDTOs(item.StoryboardShots),
		Assets:          assetDTOs(item.Assets),
		ApprovalGates:   approvalGateDTOs(item.ApprovalGates),
		GenerationJobs:  generationJobDTOs(item.GenerationJobs),
	}
}

func storyboardWorkspaceSummaryDTO(item service.StoryboardWorkspaceSummary) storyboardWorkspaceSummaryResponse {
	return storyboardWorkspaceSummaryResponse{
		AnalysisCount:             item.AnalysisCount,
		StoryMapReady:             item.StoryMapReady,
		ReadyAssetsCount:          item.ReadyAssetsCount,
		PendingApprovalGatesCount: item.PendingApprovalGatesCount,
	}
}

func storyboardWorkspaceShotDTOs(items []service.StoryboardWorkspaceShot) []storyboardWorkspaceShotResponse {
	responses := make([]storyboardWorkspaceShotResponse, 0, len(items))
	for _, item := range items {
		response := storyboardWorkspaceShotResponse{
			ID:          item.Shot.ID,
			ProjectID:   item.Shot.ProjectID,
			EpisodeID:   item.Shot.EpisodeID,
			SceneID:     item.Shot.SceneID,
			Code:        item.Shot.Code,
			Title:       item.Shot.Title,
			Description: item.Shot.Description,
			Prompt:      item.Shot.Prompt,
			Position:    item.Shot.Position,
			DurationMS:  item.Shot.DurationMS,
			CreatedAt:   item.Shot.CreatedAt,
			UpdatedAt:   item.Shot.UpdatedAt,
		}
		if item.Scene != nil {
			scene := sceneResponse(characterResponse{
				ID: item.Scene.ID, ProjectID: item.Scene.ProjectID, EpisodeID: item.Scene.EpisodeID,
				Code: item.Scene.Code, Name: item.Scene.Name, Description: item.Scene.Description,
				CreatedAt: item.Scene.CreatedAt, UpdatedAt: item.Scene.UpdatedAt,
			})
			response.Scene = &scene
		}
		if item.PromptPack != nil {
			pack := storyboardShotPromptPackSummaryResponse{
				ID: item.PromptPack.ID, ShotID: item.PromptPack.ShotID, Provider: item.PromptPack.Provider,
				Model: item.PromptPack.Model, Preset: item.PromptPack.Preset, TaskType: item.PromptPack.TaskType,
				UpdatedAt: item.PromptPack.UpdatedAt,
			}
			response.PromptPack = &pack
		}
		if item.LatestGenerationJob != nil {
			job := generationJobDTO(*item.LatestGenerationJob)
			response.LatestGenerationJob = &job
		}
		responses = append(responses, response)
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
