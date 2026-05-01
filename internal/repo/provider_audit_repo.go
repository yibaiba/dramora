package repo

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
)

// AppendProviderAuditParams 描述一次 provider 审计事件写入。
type AppendProviderAuditParams struct {
	EventID        string
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

// ProviderAuditFilter 描述 provider 审计日志查询参数。
// Actions/Capabilities 为空表示不过滤；Since/Until 为闭区间过滤；Limit 上限由调用方裁剪，Offset 用于分页。
type ProviderAuditFilter struct {
	OrganizationID string
	Actions        []string
	Capabilities   []string
	Since          *time.Time
	Until          *time.Time
	Limit          int
	Offset         int
}

// ProviderAuditPage 是审计列表的分页结果。
type ProviderAuditPage struct {
	Events  []domain.ProviderAuditEvent
	HasMore bool
}

// ProviderAuditRepository 抽象 provider 审计事件的持久化。
type ProviderAuditRepository interface {
	AppendProviderAuditEvent(ctx context.Context, params AppendProviderAuditParams) (domain.ProviderAuditEvent, error)
	ListProviderAuditEvents(ctx context.Context, filter ProviderAuditFilter) (ProviderAuditPage, error)
}

// MemoryProviderAuditRepository 是内存实现，主要服务于单测与开发环境。
type MemoryProviderAuditRepository struct {
	mu     sync.Mutex
	events []domain.ProviderAuditEvent
}

func NewMemoryProviderAuditRepository() *MemoryProviderAuditRepository {
	return &MemoryProviderAuditRepository{}
}

func (r *MemoryProviderAuditRepository) AppendProviderAuditEvent(
	_ context.Context,
	params AppendProviderAuditParams,
) (domain.ProviderAuditEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ev := domain.ProviderAuditEvent{
		ID:             params.EventID,
		OrganizationID: params.OrganizationID,
		Action:         params.Action,
		ActorUserID:    params.ActorUserID,
		ActorEmail:     params.ActorEmail,
		Capability:     params.Capability,
		ProviderType:   params.ProviderType,
		Model:          params.Model,
		Success:        params.Success,
		Message:        params.Message,
		CreatedAt:      params.CreatedAt,
	}
	r.events = append(r.events, ev)
	return ev, nil
}

func (r *MemoryProviderAuditRepository) ListProviderAuditEvents(
	_ context.Context,
	filter ProviderAuditFilter,
) (ProviderAuditPage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	matched := make([]domain.ProviderAuditEvent, 0)
	actionSet := stringSet(filter.Actions)
	capSet := stringSet(filter.Capabilities)
	for _, ev := range r.events {
		if filter.OrganizationID != "" && ev.OrganizationID != filter.OrganizationID {
			continue
		}
		if len(actionSet) > 0 && !actionSet[ev.Action] {
			continue
		}
		if len(capSet) > 0 && !capSet[ev.Capability] {
			continue
		}
		if filter.Since != nil && ev.CreatedAt.Before(*filter.Since) {
			continue
		}
		if filter.Until != nil && ev.CreatedAt.After(*filter.Until) {
			continue
		}
		matched = append(matched, ev)
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].CreatedAt.After(matched[j].CreatedAt)
	})
	limit := filter.Limit
	if limit <= 0 {
		limit = len(matched)
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(matched) {
		return ProviderAuditPage{Events: []domain.ProviderAuditEvent{}, HasMore: false}, nil
	}
	end := offset + limit
	hasMore := false
	if end < len(matched) {
		hasMore = true
	} else {
		end = len(matched)
	}
	return ProviderAuditPage{Events: append([]domain.ProviderAuditEvent(nil), matched[offset:end]...), HasMore: hasMore}, nil
}

func stringSet(in []string) map[string]bool {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]bool, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out[s] = true
		}
	}
	return out
}

// SQLiteProviderAuditRepository 是 SQLite 实现，复用 sqlite_migrations 中的 provider_audit_events 表。
type SQLiteProviderAuditRepository struct {
	db *sql.DB
}

func NewSQLiteProviderAuditRepository(db *sql.DB) *SQLiteProviderAuditRepository {
	return &SQLiteProviderAuditRepository{db: db}
}

func (r *SQLiteProviderAuditRepository) AppendProviderAuditEvent(
	ctx context.Context,
	params AppendProviderAuditParams,
) (domain.ProviderAuditEvent, error) {
	createdAt := params.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	successInt := 0
	if params.Success {
		successInt = 1
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO provider_audit_events
		(id, organization_id, action, actor_user_id, actor_email,
		 capability, provider_type, model, success, message, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		params.EventID, params.OrganizationID, params.Action,
		nullableString(params.ActorUserID), nullableString(params.ActorEmail),
		params.Capability, params.ProviderType, nullableString(params.Model),
		successInt, nullableString(params.Message),
		createdAt.UTC().Format("2006-01-02T15:04:05.000Z"),
	)
	if err != nil {
		return domain.ProviderAuditEvent{}, fmt.Errorf("append provider audit: %w", err)
	}
	return domain.ProviderAuditEvent{
		ID:             params.EventID,
		OrganizationID: params.OrganizationID,
		Action:         params.Action,
		ActorUserID:    params.ActorUserID,
		ActorEmail:     params.ActorEmail,
		Capability:     params.Capability,
		ProviderType:   params.ProviderType,
		Model:          params.Model,
		Success:        params.Success,
		Message:        params.Message,
		CreatedAt:      createdAt,
	}, nil
}

func (r *SQLiteProviderAuditRepository) ListProviderAuditEvents(
	ctx context.Context,
	filter ProviderAuditFilter,
) (ProviderAuditPage, error) {
	var (
		conds []string
		args  []any
	)
	if filter.OrganizationID != "" {
		conds = append(conds, "organization_id = ?")
		args = append(args, filter.OrganizationID)
	}
	if len(filter.Actions) > 0 {
		ph := make([]string, 0, len(filter.Actions))
		for _, a := range filter.Actions {
			ph = append(ph, "?")
			args = append(args, a)
		}
		conds = append(conds, "action IN ("+strings.Join(ph, ",")+")")
	}
	if len(filter.Capabilities) > 0 {
		ph := make([]string, 0, len(filter.Capabilities))
		for _, c := range filter.Capabilities {
			ph = append(ph, "?")
			args = append(args, c)
		}
		conds = append(conds, "capability IN ("+strings.Join(ph, ",")+")")
	}
	if filter.Since != nil {
		conds = append(conds, "created_at >= ?")
		args = append(args, filter.Since.UTC().Format("2006-01-02T15:04:05.000Z"))
	}
	if filter.Until != nil {
		conds = append(conds, "created_at <= ?")
		args = append(args, filter.Until.UTC().Format("2006-01-02T15:04:05.000Z"))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	// fetch limit+1 to detect hasMore.
	args = append(args, limit+1, offset)
	q := fmt.Sprintf(`SELECT id, organization_id, action, actor_user_id, actor_email,
		capability, provider_type, model, success, message, created_at
		FROM provider_audit_events %s
		ORDER BY created_at DESC LIMIT ? OFFSET ?`, where)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return ProviderAuditPage{}, fmt.Errorf("list provider audit: %w", err)
	}
	defer rows.Close()
	var out []domain.ProviderAuditEvent
	for rows.Next() {
		var (
			ev          domain.ProviderAuditEvent
			actorUserID sql.NullString
			actorEmail  sql.NullString
			model       sql.NullString
			message     sql.NullString
			successInt  int
			createdAt   string
		)
		if err := rows.Scan(&ev.ID, &ev.OrganizationID, &ev.Action,
			&actorUserID, &actorEmail, &ev.Capability, &ev.ProviderType,
			&model, &successInt, &message, &createdAt); err != nil {
			return ProviderAuditPage{}, fmt.Errorf("scan provider audit: %w", err)
		}
		ev.ActorUserID = actorUserID.String
		ev.ActorEmail = actorEmail.String
		ev.Model = model.String
		ev.Message = message.String
		ev.Success = successInt != 0
		if t, err := time.Parse("2006-01-02T15:04:05.000Z", createdAt); err == nil {
			ev.CreatedAt = t
		} else if t, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
			ev.CreatedAt = t
		}
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return ProviderAuditPage{}, fmt.Errorf("iterate provider audit: %w", err)
	}
	hasMore := false
	if len(out) > limit {
		hasMore = true
		out = out[:limit]
	}
	return ProviderAuditPage{Events: out, HasMore: hasMore}, nil
}
