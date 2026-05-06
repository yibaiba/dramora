package domain

import "time"

type User struct {
	ID          string
	Email       string
	DisplayName string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type OrganizationMember struct {
	OrganizationID string
	UserID         string
	Email          string
	DisplayName    string
	Role           string
	JoinedAt       time.Time
	LastActivityAt time.Time
}

const (
	APIKeyScopeReadOnly = "read-only"
	APIKeyScopeWrite    = "write"
	APIKeyScopeAdmin    = "admin"
)

type UserAPIKey struct {
	ID           string
	UserID       string
	Name         string
	TokenPreview string
	Scope        string
	IsActive     bool
	ExpiresAt    *time.Time
	LastUsedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

const (
	InvitationStatusPending  = "pending"
	InvitationStatusAccepted = "accepted"
	InvitationStatusRevoked  = "revoked"
)

type OrganizationInvitation struct {
	ID               string
	OrganizationID   string
	Email            string
	Role             string
	Token            string
	InvitedByUserID  string
	Status           string
	ExpiresAt        time.Time
	AcceptedAt       *time.Time
	AcceptedByUserID string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

const (
	InvitationActionCreated  = "created"
	InvitationActionAccepted = "accepted"
	InvitationActionRevoked  = "revoked"
	InvitationActionResent   = "resent"
)

// InvitationAuditEvent 记录一次对邀请的关键动作（创建 / 接受 / 吊销 / 重发）。
// 字段 Email/Role 是事件发生时的快照，避免邀请记录被吊销/删除后审计丢失上下文。
type InvitationAuditEvent struct {
	ID             string
	OrganizationID string
	InvitationID   string
	Action         string
	ActorUserID    string
	ActorEmail     string
	Email          string
	Role           string
	Note           string
	CreatedAt      time.Time
}
