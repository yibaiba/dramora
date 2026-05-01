package repo

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
)

type NotificationRepository interface {
	CreateNotification(ctx context.Context, n *domain.Notification) error
	ListNotifications(ctx context.Context, orgID string, userID string, filter domain.NotificationFilter) ([]domain.Notification, bool, error)
	MarkAsRead(ctx context.Context, orgID string, notificationID string, userID string) error
	MarkAllAsRead(ctx context.Context, orgID string, userID string) error
	GetUnreadCount(ctx context.Context, orgID string, userID string) (int, error)
}

// MemoryNotificationRepository implements NotificationRepository in-memory.
type MemoryNotificationRepository struct {
	mu            sync.RWMutex
	notifications map[string]*domain.Notification
}

func NewMemoryNotificationRepository() *MemoryNotificationRepository {
	return &MemoryNotificationRepository{
		notifications: make(map[string]*domain.Notification),
	}
}

func (r *MemoryNotificationRepository) CreateNotification(ctx context.Context, n *domain.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n.ID == "" {
		id, _ := domain.NewID()
	n.ID = id
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}
	r.notifications[n.ID] = n
	return nil
}

func (r *MemoryNotificationRepository) ListNotifications(ctx context.Context, orgID string, userID string, filter domain.NotificationFilter) ([]domain.Notification, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []domain.Notification
	for _, n := range r.notifications {
		if n.OrganizationID != orgID {
			continue
		}
		// Match recipient: broadcast (nil) or specific user
		if n.RecipientUserID != nil && *n.RecipientUserID != userID {
			continue
		}
		// Filter unread only
		if filter.UnreadOnly && n.ReadAt != nil {
			continue
		}
		results = append(results, *n)
	}

	// Sort by created_at desc
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].CreatedAt.After(results[i].CreatedAt) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	limit, offset := filter.Limit, filter.Offset
	if limit <= 0 {
		limit = 50
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	hasMore := end < len(results)
	if offset >= len(results) {
		return []domain.Notification{}, false, nil
	}

	return results[offset:end], hasMore, nil
}

func (r *MemoryNotificationRepository) MarkAsRead(ctx context.Context, orgID string, notificationID string, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	n, ok := r.notifications[notificationID]
	if !ok {
		return errors.New("not_found")
	}
	if n.OrganizationID != orgID {
		return errors.New("unauthorized")
	}
	if n.RecipientUserID != nil && *n.RecipientUserID != userID {
		return errors.New("unauthorized")
	}

	now := time.Now()
	n.ReadAt = &now
	return nil
}

func (r *MemoryNotificationRepository) MarkAllAsRead(ctx context.Context, orgID string, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for _, n := range r.notifications {
		if n.OrganizationID != orgID {
			continue
		}
		// Match recipient
		if n.RecipientUserID != nil && *n.RecipientUserID != userID {
			continue
		}
		n.ReadAt = &now
	}
	return nil
}

func (r *MemoryNotificationRepository) GetUnreadCount(ctx context.Context, orgID string, userID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, n := range r.notifications {
		if n.OrganizationID != orgID {
			continue
		}
		if n.RecipientUserID != nil && *n.RecipientUserID != userID {
			continue
		}
		if n.ReadAt == nil {
			count++
		}
	}
	return count, nil
}

// SQLiteNotificationRepository implements NotificationRepository for SQLite.
type SQLiteNotificationRepository struct {
	db *sql.DB
}

func NewSQLiteNotificationRepository(db *sql.DB) *SQLiteNotificationRepository {
	return &SQLiteNotificationRepository{db}
}

func (r *SQLiteNotificationRepository) CreateNotification(ctx context.Context, n *domain.Notification) error {
	if n.ID == "" {
		id, _ := domain.NewID()
	n.ID = id
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}

	meta, _ := json.Marshal(n.Metadata)

	_, err := r.db.Exec(`
		INSERT INTO notifications (id, organization_id, recipient_user_id, kind, title, body, metadata, read_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, n.ID, n.OrganizationID, n.RecipientUserID, string(n.Kind), n.Title, n.Body, string(meta), n.ReadAt, n.CreatedAt.Format(time.RFC3339))

	return err
}

func (r *SQLiteNotificationRepository) ListNotifications(ctx context.Context, orgID string, userID string, filter domain.NotificationFilter) ([]domain.Notification, bool, error) {
	limit, offset := filter.Limit, filter.Offset
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, organization_id, recipient_user_id, kind, title, body, metadata, read_at, created_at
		FROM notifications
		WHERE organization_id = ? AND (recipient_user_id IS NULL OR recipient_user_id = ?)
	`
	args := []interface{}{orgID, userID}

	if filter.UnreadOnly {
		query += ` AND read_at IS NULL`
	}

	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit+1, offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var results []domain.Notification
	for rows.Next() {
		var n domain.Notification
		var recipientUserID *string
		var metadata string
		var readAt *string

		err := rows.Scan(&n.ID, &n.OrganizationID, &recipientUserID, (*string)(&n.Kind), &n.Title, &n.Body, &metadata, &readAt, &n.CreatedAt)
		if err != nil {
			return nil, false, err
		}

		n.RecipientUserID = recipientUserID
		if readAt != nil {
			t, _ := time.Parse(time.RFC3339, *readAt)
			n.ReadAt = &t
		}
		_ = json.Unmarshal([]byte(metadata), &n.Metadata)

		results = append(results, n)
	}

	hasMore := len(results) > limit
	if hasMore {
		results = results[:limit]
	}

	return results, hasMore, nil
}

func (r *SQLiteNotificationRepository) MarkAsRead(ctx context.Context, orgID string, notificationID string, userID string) error {
	// Verify ownership
	var id string
	var recipientUserID *string
	err := r.db.QueryRow(`
		SELECT id, recipient_user_id FROM notifications
		WHERE id = ? AND organization_id = ?
	`, notificationID, orgID).Scan(&id, &recipientUserID)
	if err == sql.ErrNoRows {
		return errors.New("not_found")
	}
	if err != nil {
		return err
	}
	if recipientUserID != nil && *recipientUserID != userID {
		return errors.New("unauthorized")
	}

	now := time.Now().Format(time.RFC3339)
	_, err = r.db.Exec(`UPDATE notifications SET read_at = ? WHERE id = ?`, now, notificationID)
	return err
}

func (r *SQLiteNotificationRepository) MarkAllAsRead(ctx context.Context, orgID string, userID string) error {
	now := time.Now().Format(time.RFC3339)
	_, err := r.db.Exec(`
		UPDATE notifications
		SET read_at = ?
		WHERE organization_id = ? AND read_at IS NULL AND (recipient_user_id IS NULL OR recipient_user_id = ?)
	`, now, orgID, userID)
	return err
}

func (r *SQLiteNotificationRepository) GetUnreadCount(ctx context.Context, orgID string, userID string) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM notifications
		WHERE organization_id = ? AND read_at IS NULL AND (recipient_user_id IS NULL OR recipient_user_id = ?)
	`, orgID, userID).Scan(&count)
	return count, err
}

// PostgresNotificationRepository implements NotificationRepository for Postgres.
type PostgresNotificationRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresNotificationRepository(pool *pgxpool.Pool) *PostgresNotificationRepository {
	return &PostgresNotificationRepository{pool}
}

func (r *PostgresNotificationRepository) CreateNotification(ctx context.Context, n *domain.Notification) error {
	if n.ID == "" {
		id, _ := domain.NewID()
	n.ID = id
	}
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}

	meta, _ := json.Marshal(n.Metadata)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO notifications (id, organization_id, recipient_user_id, kind, title, body, metadata, read_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, n.ID, n.OrganizationID, n.RecipientUserID, string(n.Kind), n.Title, n.Body, meta, n.ReadAt, n.CreatedAt)

	return err
}

func (r *PostgresNotificationRepository) ListNotifications(ctx context.Context, orgID string, userID string, filter domain.NotificationFilter) ([]domain.Notification, bool, error) {
	limit, offset := filter.Limit, filter.Offset
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, organization_id, recipient_user_id, kind, title, body, metadata, read_at, created_at
		FROM notifications
		WHERE organization_id = $1 AND (recipient_user_id IS NULL OR recipient_user_id = $2)
	`
	args := []interface{}{orgID, userID}
	argNum := 3

	if filter.UnreadOnly {
		query += ` AND read_at IS NULL`
	}

	query += ` ORDER BY created_at DESC LIMIT $` + string(rune(argNum)) + ` OFFSET $` + string(rune(argNum+1))
	args = append(args, limit+1, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var results []domain.Notification
	for rows.Next() {
		var n domain.Notification
		var metadata []byte

		err := rows.Scan(&n.ID, &n.OrganizationID, &n.RecipientUserID, (*string)(&n.Kind), &n.Title, &n.Body, &metadata, &n.ReadAt, &n.CreatedAt)
		if err != nil {
			return nil, false, err
		}

		_ = json.Unmarshal(metadata, &n.Metadata)
		results = append(results, n)
	}

	hasMore := len(results) > limit
	if hasMore {
		results = results[:limit]
	}

	return results, hasMore, nil
}

func (r *PostgresNotificationRepository) MarkAsRead(ctx context.Context, orgID string, notificationID string, userID string) error {
	var id string
	var recipientUserID *string
	err := r.pool.QueryRow(ctx, `
		SELECT id, recipient_user_id FROM notifications
		WHERE id = $1 AND organization_id = $2
	`, notificationID, orgID).Scan(&id, &recipientUserID)
	if err == sql.ErrNoRows {
		return errors.New("not_found")
	}
	if err != nil {
		return err
	}
	if recipientUserID != nil && *recipientUserID != userID {
		return errors.New("unauthorized")
	}

	now := time.Now()
	_, err = r.pool.Exec(ctx, `UPDATE notifications SET read_at = $1 WHERE id = $2`, now, notificationID)
	return err
}

func (r *PostgresNotificationRepository) MarkAllAsRead(ctx context.Context, orgID string, userID string) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE notifications
		SET read_at = $1
		WHERE organization_id = $2 AND read_at IS NULL AND (recipient_user_id IS NULL OR recipient_user_id = $3)
	`, now, orgID, userID)
	return err
}

func (r *PostgresNotificationRepository) GetUnreadCount(ctx context.Context, orgID string, userID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications
		WHERE organization_id = $1 AND read_at IS NULL AND (recipient_user_id IS NULL OR recipient_user_id = $2)
	`, orgID, userID).Scan(&count)
	return count, err
}
