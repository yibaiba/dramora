package provider

import "context"

type TaskType string

const (
	TaskTypeLLM          TaskType = "llm"
	TaskTypeImage        TaskType = "image"
	TaskTypeTextToVideo  TaskType = "text_to_video"
	TaskTypeImageToVideo TaskType = "image_to_video"
	TaskTypeFirstLast    TaskType = "first_last_frame_to_video"
	TaskTypeTTS          TaskType = "tts"
	TaskTypeLipSync      TaskType = "lip_sync"
	TaskTypeAvatarVideo  TaskType = "avatar_video" // HeyGen virtual presenter
)

type Capability struct {
	TaskType           TaskType
	MaxReferenceImages int
	MaxDurationSeconds int
	SupportsCancel     bool
}

type Adapter interface {
	Name() string
	Capabilities(ctx context.Context) ([]Capability, error)
}
