package domain

import "time"

type User struct {
	ID          string
	Email       string
	DisplayName string
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
