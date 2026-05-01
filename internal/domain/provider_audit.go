package domain

import "time"

// ProviderAuditAction 枚举。
const (
	ProviderAuditActionSave  = "save"
	ProviderAuditActionTest  = "test"
	ProviderAuditActionSmoke       = "smoke"
	ProviderAuditActionSmokeStream = "smoke_stream"
)

// ProviderAuditEvent 记录管理员对 provider 端点配置的关键动作。
// Capability/ProviderType/Model 是事件发生时的快照，避免 provider 配置后续被覆写后审计上下文丢失。
// Success 标识此次动作是否成功；Message 在失败时承载原因，在成功时承载补充信息（如 probe URL）。
type ProviderAuditEvent struct {
	ID             string
	OrganizationID string
	Action         string
	ActorUserID    string
	ActorEmail     string
	Capability     string
	ProviderType   string
	Model          string
	Success        bool
	Message        string
	CreatedAt      time.Time
}
