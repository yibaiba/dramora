package domain

import "time"

type NotificationKind string

const (
	NotificationKindWalletCredit       NotificationKind = "wallet_credit"
	NotificationKindWalletDebit        NotificationKind = "wallet_debit"
	NotificationKindInvitationCreated  NotificationKind = "invitation_created"
	NotificationKindInvitationResent   NotificationKind = "invitation_resent"
	NotificationKindProviderConfigSave NotificationKind = "provider_config_save"
)

type Notification struct {
	ID              string
	OrganizationID  string
	RecipientUserID *string // nil = broadcast to all org members
	Kind            NotificationKind
	Title           string
	Body            string
	Metadata        map[string]interface{} // json serialize
	ReadAt          *time.Time
	CreatedAt       time.Time
}

type NotificationFilter struct {
	Limit      int
	Offset     int
	UnreadOnly bool
}

func IsValidNotificationKind(s string) bool {
	switch NotificationKind(s) {
	case NotificationKindWalletCredit, NotificationKindWalletDebit,
		NotificationKindInvitationCreated, NotificationKindInvitationResent,
		NotificationKindProviderConfigSave:
		return true
	}
	return false
}
