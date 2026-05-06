package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yibaiba/dramora/internal/domain"
)

type AuthIdentity struct {
	User           domain.User
	OrganizationID string
	Role           string
	PasswordHash   string
}

type CreateUserWithMembershipParams struct {
	UserID         string
	OrganizationID string
	Email          string
	DisplayName    string
	PasswordHash   string
	Role           string
}

type CreateUserAPIKeyParams struct {
	KeyID        string
	UserID       string
	Name         string
	TokenHash    string
	TokenPreview string
	Scope        string
	IsActive     bool
	ExpiresAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UpdateUserAPIKeyParams struct {
	KeyID     string
	UserID    string
	Name      string
	Scope     string
	ExpiresAt *time.Time
	UpdatedAt time.Time
}

type IdentityRepository interface {
	CreateUserWithMembership(ctx context.Context, params CreateUserWithMembershipParams) (AuthIdentity, error)
	GetAuthIdentityByEmail(ctx context.Context, email string) (AuthIdentity, error)
	GetAuthIdentityByUserID(ctx context.Context, userID string) (AuthIdentity, error)
	UpdateUserPasswordHash(ctx context.Context, userID, passwordHash string, updatedAt time.Time) error
	ListUserAPIKeys(ctx context.Context, userID string) ([]domain.UserAPIKey, error)
	CreateUserAPIKey(ctx context.Context, params CreateUserAPIKeyParams) (domain.UserAPIKey, error)
	UpdateUserAPIKey(ctx context.Context, params UpdateUserAPIKeyParams) (domain.UserAPIKey, error)
	SetUserAPIKeyActive(ctx context.Context, userID, keyID string, isActive bool, updatedAt time.Time) (domain.UserAPIKey, error)
	DeleteUserAPIKey(ctx context.Context, userID, keyID string) error
	ListOrganizationMembers(ctx context.Context, organizationID string) ([]domain.OrganizationMember, error)
	GetOrganizationMember(ctx context.Context, organizationID, userID string) (domain.OrganizationMember, error)
	UpdateOrganizationMemberRole(ctx context.Context, organizationID, userID, role string) error
	RemoveOrganizationMember(ctx context.Context, organizationID, userID string) error

	CreateOrganization(ctx context.Context, params CreateOrganizationParams) error
	CreateInvitation(ctx context.Context, params CreateInvitationParams) (domain.OrganizationInvitation, error)
	GetInvitationByToken(ctx context.Context, token string) (domain.OrganizationInvitation, error)
	MarkInvitationAccepted(ctx context.Context, invitationID, userID string, acceptedAt time.Time) error
	ListOrganizationInvitations(ctx context.Context, organizationID string) ([]domain.OrganizationInvitation, error)
	RevokeInvitation(ctx context.Context, invitationID, organizationID string, revokedAt time.Time) error

	AppendInvitationAuditEvent(ctx context.Context, params AppendInvitationAuditParams) (domain.InvitationAuditEvent, error)
	ListInvitationAuditEvents(ctx context.Context, filter InvitationAuditFilter) (InvitationAuditPage, error)
}

// InvitationAuditFilter 描述邀请审计日志查询参数。
// Actions 为空表示不过滤；Email 为空表示不过滤，比对时按小写子串匹配。
// Since/Until 为闭区间过滤；Limit 上限由调用方裁剪，Offset 用于分页。
type InvitationAuditFilter struct {
	OrganizationID string
	Actions        []string
	Email          string
	Since          *time.Time
	Until          *time.Time
	Limit          int
	Offset         int
}

// InvitationAuditPage 是审计列表的分页结果。
// Events 长度 ≤ Limit；HasMore 表示底层是否还有更多记录。
type InvitationAuditPage struct {
	Events  []domain.InvitationAuditEvent
	HasMore bool
}

type CreateOrganizationParams struct {
	OrganizationID string
	Name           string
}

type CreateInvitationParams struct {
	InvitationID    string
	OrganizationID  string
	Email           string
	Role            string
	Token           string
	InvitedByUserID string
	ExpiresAt       time.Time
}

type AppendInvitationAuditParams struct {
	EventID        string
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

type PostgresIdentityRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresIdentityRepository(pool *pgxpool.Pool) *PostgresIdentityRepository {
	return &PostgresIdentityRepository{pool: pool}
}

func (r *PostgresIdentityRepository) CreateUserWithMembership(
	ctx context.Context,
	params CreateUserWithMembershipParams,
) (AuthIdentity, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return AuthIdentity{}, fmt.Errorf("begin create user: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, createUserSQL,
		params.UserID,
		params.Email,
		params.DisplayName,
		params.PasswordHash,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return AuthIdentity{}, domain.ErrInvalidInput
		}
		if isForeignKeyViolation(err) {
			return AuthIdentity{}, domain.ErrNotFound
		}
		return AuthIdentity{}, fmt.Errorf("create user: %w", err)
	}

	_, err = tx.Exec(ctx, createOrganizationMemberSQL, params.OrganizationID, params.UserID, params.Role)
	if err != nil {
		if isUniqueViolation(err) {
			return AuthIdentity{}, domain.ErrInvalidInput
		}
		if isForeignKeyViolation(err) {
			return AuthIdentity{}, domain.ErrNotFound
		}
		return AuthIdentity{}, fmt.Errorf("create organization member: %w", err)
	}

	identity, err := scanAuthIdentity(tx.QueryRow(ctx, getAuthIdentityByUserIDSQL, params.UserID))
	if err != nil {
		return AuthIdentity{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return AuthIdentity{}, fmt.Errorf("commit create user: %w", err)
	}
	return identity, nil
}

func (r *PostgresIdentityRepository) GetAuthIdentityByEmail(ctx context.Context, email string) (AuthIdentity, error) {
	identity, err := scanAuthIdentity(r.pool.QueryRow(ctx, getAuthIdentityByEmailSQL, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthIdentity{}, domain.ErrNotFound
	}
	return identity, err
}

func (r *PostgresIdentityRepository) GetAuthIdentityByUserID(ctx context.Context, userID string) (AuthIdentity, error) {
	identity, err := scanAuthIdentity(r.pool.QueryRow(ctx, getAuthIdentityByUserIDSQL, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthIdentity{}, domain.ErrNotFound
	}
	return identity, err
}

func (r *PostgresIdentityRepository) UpdateUserPasswordHash(
	ctx context.Context,
	userID, passwordHash string,
	updatedAt time.Time,
) error {
	tag, err := r.pool.Exec(ctx, updateUserPasswordHashSQL, userID, passwordHash, updatedAt.UTC())
	if err != nil {
		return fmt.Errorf("update user password hash: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresIdentityRepository) ListUserAPIKeys(ctx context.Context, userID string) ([]domain.UserAPIKey, error) {
	rows, err := r.pool.Query(ctx, listUserAPIKeysSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("list user api keys: %w", err)
	}
	defer rows.Close()
	out := make([]domain.UserAPIKey, 0)
	for rows.Next() {
		item, scanErr := scanUserAPIKey(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan user api key: %w", scanErr)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *PostgresIdentityRepository) CreateUserAPIKey(
	ctx context.Context,
	params CreateUserAPIKeyParams,
) (domain.UserAPIKey, error) {
	item, err := scanUserAPIKey(r.pool.QueryRow(
		ctx,
		createUserAPIKeySQL,
		params.KeyID,
		params.UserID,
		params.Name,
		params.TokenHash,
		params.TokenPreview,
		params.Scope,
		params.IsActive,
		params.ExpiresAt,
		params.CreatedAt.UTC(),
		params.UpdatedAt.UTC(),
	))
	if isForeignKeyViolation(err) {
		return domain.UserAPIKey{}, domain.ErrNotFound
	}
	if isUniqueViolation(err) {
		return domain.UserAPIKey{}, domain.ErrInvalidInput
	}
	if err != nil {
		return domain.UserAPIKey{}, fmt.Errorf("create user api key: %w", err)
	}
	return item, nil
}

func (r *PostgresIdentityRepository) UpdateUserAPIKey(
	ctx context.Context,
	params UpdateUserAPIKeyParams,
) (domain.UserAPIKey, error) {
	item, err := scanUserAPIKey(r.pool.QueryRow(
		ctx,
		updateUserAPIKeySQL,
		params.KeyID,
		params.UserID,
		params.Name,
		params.Scope,
		params.ExpiresAt,
		params.UpdatedAt.UTC(),
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.UserAPIKey{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.UserAPIKey{}, fmt.Errorf("update user api key: %w", err)
	}
	return item, nil
}

func (r *PostgresIdentityRepository) SetUserAPIKeyActive(
	ctx context.Context,
	userID, keyID string,
	isActive bool,
	updatedAt time.Time,
) (domain.UserAPIKey, error) {
	item, err := scanUserAPIKey(r.pool.QueryRow(
		ctx,
		setUserAPIKeyActiveSQL,
		keyID,
		userID,
		isActive,
		updatedAt.UTC(),
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.UserAPIKey{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.UserAPIKey{}, fmt.Errorf("set user api key active: %w", err)
	}
	return item, nil
}

func (r *PostgresIdentityRepository) DeleteUserAPIKey(ctx context.Context, userID, keyID string) error {
	tag, err := r.pool.Exec(ctx, deleteUserAPIKeySQL, keyID, userID)
	if err != nil {
		return fmt.Errorf("delete user api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresIdentityRepository) ListOrganizationMembers(
	ctx context.Context,
	organizationID string,
) ([]domain.OrganizationMember, error) {
	rows, err := r.pool.Query(ctx, listOrganizationMembersSQL, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list organization members: %w", err)
	}
	defer rows.Close()
	out := make([]domain.OrganizationMember, 0)
	for rows.Next() {
		member, scanErr := scanOrganizationMember(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan organization member: %w", scanErr)
		}
		out = append(out, member)
	}
	return out, rows.Err()
}

func (r *PostgresIdentityRepository) GetOrganizationMember(
	ctx context.Context,
	organizationID, userID string,
) (domain.OrganizationMember, error) {
	member, err := scanOrganizationMember(r.pool.QueryRow(ctx, getOrganizationMemberSQL, organizationID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OrganizationMember{}, domain.ErrNotFound
	}
	return member, err
}

func (r *PostgresIdentityRepository) UpdateOrganizationMemberRole(
	ctx context.Context,
	organizationID, userID, role string,
) error {
	tag, err := r.pool.Exec(ctx, updateOrganizationMemberRoleSQL, organizationID, userID, role)
	if err != nil {
		return fmt.Errorf("update organization member role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresIdentityRepository) RemoveOrganizationMember(
	ctx context.Context,
	organizationID, userID string,
) error {
	tag, err := r.pool.Exec(ctx, removeOrganizationMemberSQL, organizationID, userID)
	if err != nil {
		return fmt.Errorf("remove organization member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

type sqliteScanner interface {
	Scan(dest ...any) error
}

func scanAuthIdentity(scanner sqliteScanner) (AuthIdentity, error) {
	var identity AuthIdentity
	err := scanner.Scan(
		&identity.User.ID,
		&identity.User.Email,
		&identity.User.DisplayName,
		&identity.PasswordHash,
		&identity.OrganizationID,
		&identity.Role,
		&identity.User.CreatedAt,
		&identity.User.UpdatedAt,
	)
	return identity, err
}

func scanOrganizationMember(scanner sqliteScanner) (domain.OrganizationMember, error) {
	var member domain.OrganizationMember
	err := scanner.Scan(
		&member.OrganizationID,
		&member.UserID,
		&member.Email,
		&member.DisplayName,
		&member.Role,
		&member.JoinedAt,
		&member.LastActivityAt,
	)
	if err != nil {
		return domain.OrganizationMember{}, err
	}
	member.JoinedAt = member.JoinedAt.UTC()
	member.LastActivityAt = member.LastActivityAt.UTC()
	return member, nil
}

func scanUserAPIKey(scanner sqliteScanner) (domain.UserAPIKey, error) {
	var item domain.UserAPIKey
	var expiresAt sql.NullTime
	var lastUsedAt sql.NullTime
	err := scanner.Scan(
		&item.ID,
		&item.UserID,
		&item.Name,
		&item.TokenPreview,
		&item.Scope,
		&item.IsActive,
		&expiresAt,
		&lastUsedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return domain.UserAPIKey{}, err
	}
	item.CreatedAt = item.CreatedAt.UTC()
	item.UpdatedAt = item.UpdatedAt.UTC()
	if expiresAt.Valid {
		t := expiresAt.Time.UTC()
		item.ExpiresAt = &t
	}
	if lastUsedAt.Valid {
		t := lastUsedAt.Time.UTC()
		item.LastUsedAt = &t
	}
	return item, nil
}

func (r *PostgresIdentityRepository) CreateOrganization(ctx context.Context, params CreateOrganizationParams) error {
	if _, err := r.pool.Exec(ctx, createOrganizationSQL, params.OrganizationID, params.Name); err != nil {
		if isUniqueViolation(err) {
			return domain.ErrInvalidInput
		}
		return fmt.Errorf("create organization: %w", err)
	}
	return nil
}

func (r *PostgresIdentityRepository) CreateInvitation(
	ctx context.Context,
	params CreateInvitationParams,
) (domain.OrganizationInvitation, error) {
	var invitedBy any
	if params.InvitedByUserID != "" {
		invitedBy = params.InvitedByUserID
	}
	row := r.pool.QueryRow(ctx, createInvitationSQL,
		params.InvitationID,
		params.OrganizationID,
		params.Email,
		params.Role,
		params.Token,
		invitedBy,
		params.ExpiresAt,
	)
	inv, err := scanInvitation(row)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.OrganizationInvitation{}, domain.ErrInvalidInput
		}
		if isForeignKeyViolation(err) {
			return domain.OrganizationInvitation{}, domain.ErrNotFound
		}
		return domain.OrganizationInvitation{}, fmt.Errorf("create invitation: %w", err)
	}
	return inv, nil
}

func (r *PostgresIdentityRepository) GetInvitationByToken(ctx context.Context, token string) (domain.OrganizationInvitation, error) {
	inv, err := scanInvitation(r.pool.QueryRow(ctx, getInvitationByTokenSQL, token))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.OrganizationInvitation{}, domain.ErrNotFound
	}
	return inv, err
}

func (r *PostgresIdentityRepository) MarkInvitationAccepted(ctx context.Context, invitationID, userID string, acceptedAt time.Time) error {
	tag, err := r.pool.Exec(ctx, markInvitationAcceptedSQL, invitationID, userID, acceptedAt)
	if err != nil {
		return fmt.Errorf("mark invitation accepted: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresIdentityRepository) ListOrganizationInvitations(ctx context.Context, organizationID string) ([]domain.OrganizationInvitation, error) {
	rows, err := r.pool.Query(ctx, listInvitationsByOrgSQL, organizationID)
	if err != nil {
		return nil, fmt.Errorf("list invitations: %w", err)
	}
	defer rows.Close()
	var out []domain.OrganizationInvitation
	for rows.Next() {
		inv, scanErr := scanInvitation(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan invitation: %w", scanErr)
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (r *PostgresIdentityRepository) RevokeInvitation(ctx context.Context, invitationID, organizationID string, revokedAt time.Time) error {
	tag, err := r.pool.Exec(ctx, revokeInvitationSQL, invitationID, organizationID, revokedAt.UTC())
	if err != nil {
		return fmt.Errorf("revoke invitation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *PostgresIdentityRepository) AppendInvitationAuditEvent(
	ctx context.Context,
	params AppendInvitationAuditParams,
) (domain.InvitationAuditEvent, error) {
	createdAt := params.CreatedAt.UTC()
	_, err := r.pool.Exec(ctx, insertInvitationAuditEventSQL,
		params.EventID,
		params.OrganizationID,
		params.InvitationID,
		params.Action,
		params.ActorUserID,
		params.ActorEmail,
		params.Email,
		params.Role,
		params.Note,
		createdAt,
	)
	if err != nil {
		return domain.InvitationAuditEvent{}, fmt.Errorf("append invitation audit: %w", err)
	}
	return domain.InvitationAuditEvent{
		ID:             params.EventID,
		OrganizationID: params.OrganizationID,
		InvitationID:   params.InvitationID,
		Action:         params.Action,
		ActorUserID:    params.ActorUserID,
		ActorEmail:     params.ActorEmail,
		Email:          params.Email,
		Role:           params.Role,
		Note:           params.Note,
		CreatedAt:      createdAt,
	}, nil
}

func (r *PostgresIdentityRepository) ListInvitationAuditEvents(
	ctx context.Context,
	filter InvitationAuditFilter,
) (InvitationAuditPage, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	args := []any{filter.OrganizationID}
	clauses := []string{"organization_id = $1::uuid"}
	if len(filter.Actions) > 0 {
		args = append(args, filter.Actions)
		clauses = append(clauses, fmt.Sprintf("action = ANY($%d::text[])", len(args)))
	}
	if email := strings.ToLower(strings.TrimSpace(filter.Email)); email != "" {
		args = append(args, "%"+email+"%")
		clauses = append(clauses, fmt.Sprintf("lower(email) LIKE $%d", len(args)))
	}
	if filter.Since != nil {
		args = append(args, filter.Since.UTC())
		clauses = append(clauses, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if filter.Until != nil {
		args = append(args, filter.Until.UTC())
		clauses = append(clauses, fmt.Sprintf("created_at <= $%d", len(args)))
	}
	args = append(args, limit+1, offset)
	query := fmt.Sprintf(
		`SELECT id, organization_id, invitation_id, action,
            actor_user_id, actor_email, email, role, note, created_at
         FROM organization_invitation_events
         WHERE %s
         ORDER BY created_at DESC
         LIMIT $%d OFFSET $%d`,
		strings.Join(clauses, " AND "),
		len(args)-1,
		len(args),
	)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return InvitationAuditPage{}, fmt.Errorf("list invitation audit: %w", err)
	}
	defer rows.Close()
	var out []domain.InvitationAuditEvent
	for rows.Next() {
		var ev domain.InvitationAuditEvent
		var actorUser, actorEmail, note *string
		if scanErr := rows.Scan(
			&ev.ID,
			&ev.OrganizationID,
			&ev.InvitationID,
			&ev.Action,
			&actorUser,
			&actorEmail,
			&ev.Email,
			&ev.Role,
			&note,
			&ev.CreatedAt,
		); scanErr != nil {
			return InvitationAuditPage{}, fmt.Errorf("scan invitation audit: %w", scanErr)
		}
		if actorUser != nil {
			ev.ActorUserID = *actorUser
		}
		if actorEmail != nil {
			ev.ActorEmail = *actorEmail
		}
		if note != nil {
			ev.Note = *note
		}
		ev.CreatedAt = ev.CreatedAt.UTC()
		out = append(out, ev)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return InvitationAuditPage{}, rowsErr
	}
	hasMore := false
	if len(out) > limit {
		out = out[:limit]
		hasMore = true
	}
	return InvitationAuditPage{Events: out, HasMore: hasMore}, nil
}

func scanInvitation(scanner sqliteScanner) (domain.OrganizationInvitation, error) {
	var inv domain.OrganizationInvitation
	var invitedBy *string
	var acceptedAt *time.Time
	var acceptedBy *string
	err := scanner.Scan(
		&inv.ID,
		&inv.OrganizationID,
		&inv.Email,
		&inv.Role,
		&inv.Token,
		&invitedBy,
		&inv.Status,
		&inv.ExpiresAt,
		&acceptedAt,
		&acceptedBy,
		&inv.CreatedAt,
		&inv.UpdatedAt,
	)
	if err != nil {
		return domain.OrganizationInvitation{}, err
	}
	if invitedBy != nil {
		inv.InvitedByUserID = *invitedBy
	}
	if acceptedAt != nil {
		t := acceptedAt.UTC()
		inv.AcceptedAt = &t
	}
	if acceptedBy != nil {
		inv.AcceptedByUserID = *acceptedBy
	}
	return inv, nil
}
