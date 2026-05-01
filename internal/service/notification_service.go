package service

import (
	"context"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

type NotificationService struct {
	notifRepo repo.NotificationRepository
}

func NewNotificationService(notifRepo repo.NotificationRepository) *NotificationService {
	return &NotificationService{notifRepo}
}

func (s *NotificationService) CreateNotification(ctx context.Context, orgID string, kind domain.NotificationKind, title string, body string, recipientUserID *string, metadata map[string]interface{}) (*domain.Notification, error) {
	n := &domain.Notification{
		OrganizationID:  orgID,
		RecipientUserID: recipientUserID,
		Kind:            kind,
		Title:           title,
		Body:            body,
		Metadata:        metadata,
	}
	if err := s.notifRepo.CreateNotification(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

func (s *NotificationService) ListNotifications(ctx context.Context, auth RequestAuthContext, limit int, offset int, unreadOnly bool) ([]domain.Notification, bool, error) {
	if auth.UserID == "" || auth.OrganizationID == "" {
		return nil, false, ErrUnauthorized
	}

	return s.notifRepo.ListNotifications(ctx, auth.OrganizationID, auth.UserID, domain.NotificationFilter{
		Limit:      limit,
		Offset:     offset,
		UnreadOnly: unreadOnly,
	})
}

func (s *NotificationService) MarkAsRead(ctx context.Context, auth RequestAuthContext, notificationID string) error {
	if auth.UserID == "" || auth.OrganizationID == "" {
		return ErrUnauthorized
	}
	return s.notifRepo.MarkAsRead(ctx, auth.OrganizationID, notificationID, auth.UserID)
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, auth RequestAuthContext) error {
	if auth.UserID == "" || auth.OrganizationID == "" {
		return ErrUnauthorized
	}
	return s.notifRepo.MarkAllAsRead(ctx, auth.OrganizationID, auth.UserID)
}

func (s *NotificationService) GetUnreadCount(ctx context.Context, auth RequestAuthContext) (int, error) {
	if auth.UserID == "" || auth.OrganizationID == "" {
		return 0, ErrUnauthorized
	}
	return s.notifRepo.GetUnreadCount(ctx, auth.OrganizationID, auth.UserID)
}
