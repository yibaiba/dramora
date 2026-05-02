package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/service"
)

type NotificationDTO struct {
	ID              string                 `json:"id"`
	OrganizationID  string                 `json:"organization_id"`
	RecipientUserID *string                `json:"recipient_user_id"`
	Kind            string                 `json:"kind"`
	Title           string                 `json:"title"`
	Body            string                 `json:"body"`
	Metadata        map[string]interface{} `json:"metadata"`
	ReadAt          *string                `json:"read_at"`
	CreatedAt       string                 `json:"created_at"`
}

func toNotificationDTO(n domain.Notification) NotificationDTO {
	var readAt *string
	if n.ReadAt != nil {
		s := n.ReadAt.UTC().Format("2006-01-02T15:04:05.000Z")
		readAt = &s
	}
	if n.Metadata == nil {
		n.Metadata = make(map[string]interface{})
	}
	return NotificationDTO{
		ID:              n.ID,
		OrganizationID:  n.OrganizationID,
		RecipientUserID: n.RecipientUserID,
		Kind:            string(n.Kind),
		Title:           n.Title,
		Body:            n.Body,
		Metadata:        n.Metadata,
		ReadAt:          readAt,
		CreatedAt:       n.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
	}
}

type CreateNotificationRequest struct {
	RecipientUserID *string                `json:"recipient_user_id"`
	Title           string                 `json:"title"`
	Body            string                 `json:"body"`
	Kind            string                 `json:"kind"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

func (api *api) createNotification(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, ok := service.RequestAuthFromContext(ctx)
	if !ok || auth.UserID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	// 检查权限：需要系统或管理员身份
	if auth.Role != "system" && auth.Role != "owner" && auth.Role != "admin" {
		writeError(w, http.StatusForbidden, "forbidden", "insufficient permissions to create notifications")
		return
	}

	var request CreateNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	// 验证必需字段
	if request.Title == "" || request.Body == "" || request.Kind == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "title, body, and kind are required")
		return
	}

	// 验证 Kind 是否有效
	if !domain.IsValidNotificationKind(request.Kind) {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid notification kind")
		return
	}

	notification, err := api.notificationService.CreateNotification(
		ctx,
		auth.OrganizationID,
		domain.NotificationKind(request.Kind),
		request.Title,
		request.Body,
		request.RecipientUserID,
		request.Metadata,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]NotificationDTO{"notification": toNotificationDTO(*notification)})
}

func (api *api) listNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, ok := service.RequestAuthFromContext(ctx)
	if !ok || auth.UserID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	unreadOnly := r.URL.Query().Get("unread_only") == "true"
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	notifications, hasMore, err := api.notificationService.ListNotifications(ctx, auth, limit, offset, unreadOnly)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	unreadCount, _ := api.notificationService.GetUnreadCount(ctx, auth)

	notificationDTOs := make([]NotificationDTO, len(notifications))
	for i, n := range notifications {
		notificationDTOs[i] = toNotificationDTO(n)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"notifications": notificationDTOs,
		"has_more":      hasMore,
		"unread_count":  unreadCount,
	})
}

func (api *api) markNotificationAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, ok := service.RequestAuthFromContext(ctx)
	if !ok || auth.UserID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	notificationID := chi.URLParam(r, "id")
	err := api.notificationService.MarkAsRead(ctx, auth, notificationID)
	if err != nil {
		if err.Error() == "not_found" {
			writeError(w, http.StatusNotFound, "not_found", "notification not found")
		} else if err.Error() == "unauthorized" {
			writeError(w, http.StatusForbidden, "forbidden", "cannot access this notification")
		} else {
			writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (api *api) markAllNotificationsAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	auth, ok := service.RequestAuthFromContext(ctx)
	if !ok || auth.UserID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	err := api.notificationService.MarkAllAsRead(ctx, auth)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
