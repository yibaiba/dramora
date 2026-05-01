package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/repo"
)

var ErrUnauthorized = errors.New("unauthorized")

const (
	defaultAccessTokenTTL  = 1 * time.Hour
	defaultRefreshTokenTTL = 30 * 24 * time.Hour
)

type AuthSession struct {
	Token            string
	User             domain.User
	OrganizationID   string
	Role             string
	ExpiresAt        time.Time
	RefreshToken     string
	RefreshTokenID   string
	RefreshExpiresAt time.Time
}

type RegisterInput struct {
	Email           string
	DisplayName     string
	Password        string
	InvitationToken string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthService struct {
	identityRepo repo.IdentityRepository
	refreshRepo  repo.RefreshTokenRepository
	jwtSecret    []byte
	accessTTL    time.Duration
	refreshTTL   time.Duration
	notificationSvc *NotificationService
}

// NewAuthService 构建认证服务。
// 不再依赖 defaultOrganizationID：注册时若无邀请令牌，AuthService 会为新
// 用户自动建立一个 owned workspace；带邀请令牌时按邀请的 org/role 加入。
//
// refreshRepo 为 nil 时退化为单 token 模式（仅签发短 access token，不发
// refresh token）。生产路径应通过 SetRefreshTokenRepository 注入实现。
func NewAuthService(identityRepo repo.IdentityRepository, jwtSecret string, notifSvc *NotificationService) *AuthService {
	return &AuthService{
		identityRepo:    identityRepo,
		jwtSecret:       []byte(jwtSecret),
		accessTTL:       defaultAccessTokenTTL,
		refreshTTL:      defaultRefreshTokenTTL,
		notificationSvc: notifSvc,
	}
}

// SetRefreshTokenRepository 注入 refresh token 存储；调用后 register/login 等会
// 自动签发 refresh token，并支持 Refresh / Logout 流程。
func (s *AuthService) SetRefreshTokenRepository(refreshRepo repo.RefreshTokenRepository) {
	s.refreshRepo = refreshRepo
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (AuthSession, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	displayName := strings.TrimSpace(input.DisplayName)
	password := strings.TrimSpace(input.Password)
	invitationToken := strings.TrimSpace(input.InvitationToken)
	if email == "" || displayName == "" || password == "" {
		return AuthSession{}, fmt.Errorf("email, display_name, and password are required: %w", domain.ErrInvalidInput)
	}
	if len(password) < 8 {
		return AuthSession{}, fmt.Errorf("password must be at least 8 characters: %w", domain.ErrInvalidInput)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return AuthSession{}, fmt.Errorf("hash password: %w", err)
	}

	organizationID, role, invitationID, err := s.resolveRegistrationOrg(ctx, displayName, email, invitationToken)
	if err != nil {
		return AuthSession{}, err
	}

	userID := uuid.NewString()
	identity, err := s.identityRepo.CreateUserWithMembership(ctx, repo.CreateUserWithMembershipParams{
		UserID:         userID,
		OrganizationID: organizationID,
		Email:          email,
		DisplayName:    displayName,
		PasswordHash:   string(passwordHash),
		Role:           role,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return AuthSession{}, fmt.Errorf("email is already registered: %w", domain.ErrInvalidInput)
		}
		return AuthSession{}, err
	}

	if invitationID != "" {
		if err := s.identityRepo.MarkInvitationAccepted(ctx, invitationID, userID, time.Now().UTC()); err != nil {
			return AuthSession{}, fmt.Errorf("mark invitation accepted: %w", err)
		}
		if _, auditErr := s.identityRepo.AppendInvitationAuditEvent(ctx, repo.AppendInvitationAuditParams{
			EventID:        uuid.NewString(),
			OrganizationID: organizationID,
			InvitationID:   invitationID,
			Action:         domain.InvitationActionAccepted,
			ActorUserID:    userID,
			ActorEmail:     email,
			Email:          email,
			Role:           role,
			CreatedAt:      time.Now().UTC(),
		}); auditErr != nil {
			return AuthSession{}, fmt.Errorf("audit invitation accepted: %w", auditErr)
		}
	}

	return s.buildSession(ctx, identity)
}

// resolveRegistrationOrg decides which organization a new user joins:
//   - With a valid pending invitation token: join that org with the invitation's role.
//   - Without a token: provision a fresh organization owned by the new user
//     (rather than dumping every new user into defaultOrganizationID).
func (s *AuthService) resolveRegistrationOrg(
	ctx context.Context,
	displayName, email, invitationToken string,
) (orgID, role, invitationID string, err error) {
	if invitationToken != "" {
		inv, lookupErr := s.identityRepo.GetInvitationByToken(ctx, invitationToken)
		if lookupErr != nil {
			if errors.Is(lookupErr, domain.ErrNotFound) {
				return "", "", "", fmt.Errorf("invitation token is invalid: %w", domain.ErrInvalidInput)
			}
			return "", "", "", lookupErr
		}
		if inv.Status != domain.InvitationStatusPending {
			return "", "", "", fmt.Errorf("invitation is no longer pending: %w", domain.ErrInvalidInput)
		}
		if !inv.ExpiresAt.IsZero() && time.Now().UTC().After(inv.ExpiresAt) {
			return "", "", "", fmt.Errorf("invitation has expired: %w", domain.ErrInvalidInput)
		}
		if inv.Email != "" && inv.Email != email {
			return "", "", "", fmt.Errorf("invitation was issued to a different email: %w", domain.ErrInvalidInput)
		}
		return inv.OrganizationID, inv.Role, inv.ID, nil
	}

	// No invitation: create a fresh workspace owned by this user.
	newOrgID := uuid.NewString()
	orgName := strings.TrimSpace(displayName) + "'s Workspace"
	if err := s.identityRepo.CreateOrganization(ctx, repo.CreateOrganizationParams{
		OrganizationID: newOrgID,
		Name:           orgName,
	}); err != nil {
		return "", "", "", fmt.Errorf("create organization: %w", err)
	}
	return newOrgID, "owner", "", nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (AuthSession, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	password := strings.TrimSpace(input.Password)
	if email == "" || password == "" {
		return AuthSession{}, fmt.Errorf("email and password are required: %w", domain.ErrInvalidInput)
	}

	identity, err := s.identityRepo.GetAuthIdentityByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return AuthSession{}, ErrUnauthorized
		}
		return AuthSession{}, err
	}
	if compareErr := bcrypt.CompareHashAndPassword([]byte(identity.PasswordHash), []byte(password)); compareErr != nil {
		return AuthSession{}, ErrUnauthorized
	}

	return s.buildSession(ctx, identity)
}

func (s *AuthService) CurrentSession(ctx context.Context, bearerToken string) (AuthSession, error) {
	token := parseBearerToken(bearerToken)
	if token == "" {
		return AuthSession{}, ErrUnauthorized
	}

	claims, err := s.verifyToken(token)
	if err != nil {
		return AuthSession{}, ErrUnauthorized
	}
	identity, err := s.identityRepo.GetAuthIdentityByUserID(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return AuthSession{}, ErrUnauthorized
		}
		return AuthSession{}, err
	}
	expiresAt := time.Unix(claims.ExpiresAt, 0).UTC()
	return AuthSession{
		Token:          token,
		User:           identity.User,
		OrganizationID: identity.OrganizationID,
		Role:           identity.Role,
		ExpiresAt:      expiresAt,
	}, nil
}

func (s *AuthService) buildSession(ctx context.Context, identity repo.AuthIdentity) (AuthSession, error) {
	expiresAt := time.Now().UTC().Add(s.accessTTL)
	token, err := s.signToken(authClaims{
		Subject:        identity.User.ID,
		OrganizationID: identity.OrganizationID,
		Role:           identity.Role,
		ExpiresAt:      expiresAt.Unix(),
	})
	if err != nil {
		return AuthSession{}, err
	}
	session := AuthSession{
		Token:          token,
		User:           identity.User,
		OrganizationID: identity.OrganizationID,
		Role:           identity.Role,
		ExpiresAt:      expiresAt,
	}
	if s.refreshRepo != nil {
		raw, refreshID, refreshExpiresAt, err := s.issueRefreshToken(ctx, identity)
		if err != nil {
			return AuthSession{}, err
		}
		session.RefreshToken = raw
		session.RefreshTokenID = refreshID
		session.RefreshExpiresAt = refreshExpiresAt
	}
	return session, nil
}

func (s *AuthService) issueRefreshToken(ctx context.Context, identity repo.AuthIdentity) (string, string, time.Time, error) {
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate refresh token: %w", err)
	}
	raw := base64.RawURLEncoding.EncodeToString(rawBytes)
	hash := hashRefreshToken(raw)
	expiresAt := time.Now().UTC().Add(s.refreshTTL)
	id := uuid.NewString()
	if _, err := s.refreshRepo.Create(ctx, repo.CreateRefreshTokenParams{
		ID:             id,
		UserID:         identity.User.ID,
		OrganizationID: identity.OrganizationID,
		Role:           identity.Role,
		TokenHash:      hash,
		ExpiresAt:      expiresAt,
	}); err != nil {
		return "", "", time.Time{}, fmt.Errorf("persist refresh token: %w", err)
	}
	return raw, id, expiresAt, nil
}

func hashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// Refresh 使用 refresh token 兑换新的 access + refresh token，旧的 refresh
// token 立即被吊销并指向新 token。
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (AuthSession, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" || s.refreshRepo == nil {
		return AuthSession{}, ErrUnauthorized
	}
	hash := hashRefreshToken(refreshToken)
	rec, err := s.refreshRepo.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return AuthSession{}, ErrUnauthorized
		}
		return AuthSession{}, err
	}
	if rec.RevokedAt != nil {
		return AuthSession{}, ErrUnauthorized
	}
	if !rec.ExpiresAt.IsZero() && time.Now().UTC().After(rec.ExpiresAt) {
		return AuthSession{}, ErrUnauthorized
	}

	identity, err := s.identityRepo.GetAuthIdentityByUserID(ctx, rec.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return AuthSession{}, ErrUnauthorized
		}
		return AuthSession{}, err
	}
	// 保留 token 颁发时的 org/role，避免在 multi-org 切换时静默升降权。
	identity.OrganizationID = rec.OrganizationID
	identity.Role = rec.Role

	session, err := s.buildSession(ctx, identity)
	if err != nil {
		return AuthSession{}, err
	}
	if session.RefreshToken == "" {
		return AuthSession{}, ErrUnauthorized
	}
	newID := hashRefreshToken(session.RefreshToken)
	// replaced_by_id 写入的是新 token 的 hash 前缀（用于审计追溯），但更稳妥的做
	// 法是写新 token 的行 id；为此我们重新查一次。
	newRec, err := s.refreshRepo.GetByHash(ctx, newID)
	if err == nil {
		if revokeErr := s.refreshRepo.Revoke(ctx, rec.ID, &newRec.ID); revokeErr != nil {
			return AuthSession{}, fmt.Errorf("revoke previous refresh token: %w", revokeErr)
		}
	} else {
		if revokeErr := s.refreshRepo.Revoke(ctx, rec.ID, nil); revokeErr != nil {
			return AuthSession{}, fmt.Errorf("revoke previous refresh token: %w", revokeErr)
		}
	}
	return session, nil
}

// Logout 主动吊销 refresh token。token 不存在或已吊销时静默成功，避免泄露存在性。
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" || s.refreshRepo == nil {
		return nil
	}
	hash := hashRefreshToken(refreshToken)
	rec, err := s.refreshRepo.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}
	if err := s.refreshRepo.Revoke(ctx, rec.ID, nil); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil
		}
		return err
	}
	return nil
}

// IssueSessionForIdentity signs a session token for an already-resolved
// identity. Useful for seeding tests or service-to-service flows where
// password validation has already happened (or doesn't apply).
func (s *AuthService) IssueSessionForIdentity(identity repo.AuthIdentity) (AuthSession, error) {
	return s.buildSession(context.Background(), identity)
}

type authClaims struct {
	Subject        string `json:"sub"`
	OrganizationID string `json:"org_id"`
	Role           string `json:"role"`
	ExpiresAt      int64  `json:"exp"`
}

func (s *AuthService) signToken(claims authClaims) (string, error) {
	header, err := json.Marshal(map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal jwt payload: %w", err)
	}

	signingInput := encodeJWTPart(header) + "." + encodeJWTPart(payload)
	mac := hmac.New(sha256.New, s.jwtSecret)
	if _, err := mac.Write([]byte(signingInput)); err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	signature := encodeJWTPart(mac.Sum(nil))
	return signingInput + "." + signature, nil
}

func (s *AuthService) verifyToken(token string) (authClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return authClaims{}, ErrUnauthorized
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, s.jwtSecret)
	if _, err := mac.Write([]byte(signingInput)); err != nil {
		return authClaims{}, fmt.Errorf("sign jwt: %w", err)
	}
	expectedSignature := encodeJWTPart(mac.Sum(nil))
	if !hmac.Equal([]byte(expectedSignature), []byte(parts[2])) {
		return authClaims{}, ErrUnauthorized
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return authClaims{}, ErrUnauthorized
	}
	var claims authClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return authClaims{}, ErrUnauthorized
	}
	if claims.Subject == "" || claims.OrganizationID == "" || claims.ExpiresAt == 0 {
		return authClaims{}, ErrUnauthorized
	}
	if time.Now().UTC().After(time.Unix(claims.ExpiresAt, 0).UTC()) {
		return authClaims{}, ErrUnauthorized
	}
	return claims, nil
}

func encodeJWTPart(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}

func parseBearerToken(value string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(value, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(value, prefix))
}

const defaultInvitationTTL = 14 * 24 * time.Hour

type CreateInvitationInput struct {
	Email string
	Role  string
}

func (s *AuthService) CreateInvitation(ctx context.Context, input CreateInvitationInput) (domain.OrganizationInvitation, error) {
	auth, ok := RequestAuthFromContext(ctx)
	if !ok || auth.OrganizationID == "" {
		return domain.OrganizationInvitation{}, ErrUnauthorized
	}
	email := strings.TrimSpace(strings.ToLower(input.Email))
	role := strings.TrimSpace(strings.ToLower(input.Role))
	if email == "" {
		return domain.OrganizationInvitation{}, fmt.Errorf("email is required: %w", domain.ErrInvalidInput)
	}
	if role == "" {
		role = "editor"
	}
	switch role {
	case "owner", "admin", "editor", "viewer":
	default:
		return domain.OrganizationInvitation{}, fmt.Errorf("role must be one of owner/admin/editor/viewer: %w", domain.ErrInvalidInput)
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return domain.OrganizationInvitation{}, fmt.Errorf("generate invitation token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)

	inv, err := s.identityRepo.CreateInvitation(ctx, repo.CreateInvitationParams{
		InvitationID:    uuid.NewString(),
		OrganizationID:  auth.OrganizationID,
		Email:           email,
		Role:            role,
		Token:           token,
		InvitedByUserID: auth.UserID,
		ExpiresAt:       time.Now().UTC().Add(defaultInvitationTTL),
	})
	if err != nil {
		return domain.OrganizationInvitation{}, err
	}
	if _, auditErr := s.identityRepo.AppendInvitationAuditEvent(ctx, repo.AppendInvitationAuditParams{
		EventID:        uuid.NewString(),
		OrganizationID: auth.OrganizationID,
		InvitationID:   inv.ID,
		Action:         domain.InvitationActionCreated,
		ActorUserID:    auth.UserID,
		ActorEmail:     "",
		Email:          inv.Email,
		Role:           inv.Role,
		CreatedAt:      time.Now().UTC(),
	}); auditErr != nil {
		return domain.OrganizationInvitation{}, fmt.Errorf("audit invitation created: %w", auditErr)
	}
	
	// Create notification for invitation
	if s.notificationSvc != nil {
		_, _ = s.notificationSvc.CreateNotification(ctx, auth.OrganizationID, domain.NotificationKindInvitationCreated, fmt.Sprintf("新增邀请：%s", inv.Email), fmt.Sprintf("角色：%s，有效期 14 天", inv.Role), nil, map[string]interface{}{
			"invitation_id":   inv.ID,
			"email":           inv.Email,
			"role":            inv.Role,
			"invited_by_user": auth.UserID,
		})
	}
	
	return inv, nil
}

func (s *AuthService) ListInvitations(ctx context.Context) ([]domain.OrganizationInvitation, error) {
	auth, ok := RequestAuthFromContext(ctx)
	if !ok || auth.OrganizationID == "" {
		return nil, ErrUnauthorized
	}
	return s.identityRepo.ListOrganizationInvitations(ctx, auth.OrganizationID)
}

// ListInvitationAuditEvents 返回当前组织邀请审计事件，按 created_at 倒序。
// 接受 actions / email / since / until / limit / offset 过滤，由 RequestAuth 强制 organization_id。
func (s *AuthService) ListInvitationAuditEvents(ctx context.Context, filter repo.InvitationAuditFilter) (repo.InvitationAuditPage, error) {
	auth, ok := RequestAuthFromContext(ctx)
	if !ok || auth.OrganizationID == "" {
		return repo.InvitationAuditPage{}, ErrUnauthorized
	}
	filter.OrganizationID = auth.OrganizationID
	return s.identityRepo.ListInvitationAuditEvents(ctx, filter)
}

// RevokeInvitation 把 pending 状态的邀请置为 revoked。仅当前组织且 pending 才能命中；
// 其余情况（不存在 / 跨组织 / 已 accepted / 已 revoked）一律返回 ErrNotFound 以便
// HTTP 层映射为 404，避免泄露邀请 ID 是否存在。
func (s *AuthService) RevokeInvitation(ctx context.Context, invitationID string) error {
	auth, ok := RequestAuthFromContext(ctx)
	if !ok || auth.OrganizationID == "" {
		return ErrUnauthorized
	}
	if strings.TrimSpace(invitationID) == "" {
		return fmt.Errorf("invitation id is required: %w", domain.ErrInvalidInput)
	}
	list, err := s.identityRepo.ListOrganizationInvitations(ctx, auth.OrganizationID)
	if err != nil {
		return err
	}
	var snap *domain.OrganizationInvitation
	for i := range list {
		if list[i].ID == invitationID {
			snap = &list[i]
			break
		}
	}
	if err := s.identityRepo.RevokeInvitation(ctx, invitationID, auth.OrganizationID, time.Now().UTC()); err != nil {
		return err
	}
	if snap != nil {
		if _, auditErr := s.identityRepo.AppendInvitationAuditEvent(ctx, repo.AppendInvitationAuditParams{
			EventID:        uuid.NewString(),
			OrganizationID: auth.OrganizationID,
			InvitationID:   invitationID,
			Action:         domain.InvitationActionRevoked,
			ActorUserID:    auth.UserID,
			ActorEmail:     "",
			Email:          snap.Email,
			Role:           snap.Role,
			CreatedAt:      time.Now().UTC(),
		}); auditErr != nil {
			return fmt.Errorf("audit invitation revoked: %w", auditErr)
		}
	}
	return nil
}

// ResendInvitation 把当前 pending 的邀请吊销并签发一条新的同 email/role 邀请，
// 用于「邀请链接丢失 / 即将过期 / 想换一个 token」的场景。仅 owner/admin 可调用，
// 命中条件与 RevokeInvitation 相同；找不到 / 跨组织 / 已 accepted / 已 revoked 都
// 返回 ErrNotFound。新邀请会带新 token 与新 expires_at。
func (s *AuthService) ResendInvitation(ctx context.Context, invitationID string) (domain.OrganizationInvitation, error) {
	auth, ok := RequestAuthFromContext(ctx)
	if !ok || auth.OrganizationID == "" {
		return domain.OrganizationInvitation{}, ErrUnauthorized
	}
	if strings.TrimSpace(invitationID) == "" {
		return domain.OrganizationInvitation{}, fmt.Errorf("invitation id is required: %w", domain.ErrInvalidInput)
	}
	list, err := s.identityRepo.ListOrganizationInvitations(ctx, auth.OrganizationID)
	if err != nil {
		return domain.OrganizationInvitation{}, err
	}
	var src *domain.OrganizationInvitation
	for i := range list {
		if list[i].ID == invitationID && list[i].Status == domain.InvitationStatusPending {
			src = &list[i]
			break
		}
	}
	if src == nil {
		return domain.OrganizationInvitation{}, domain.ErrNotFound
	}
	if err := s.RevokeInvitation(ctx, invitationID); err != nil {
		return domain.OrganizationInvitation{}, err
	}
	return s.CreateInvitation(ctx, CreateInvitationInput{Email: src.Email, Role: src.Role})
}

// SessionInfo 是 refresh token 面向调用方的脱敏视图，token_hash 已剥离。
type SessionInfo struct {
	ID             string
	OrganizationID string
	Role           string
	CreatedAt      time.Time
	ExpiresAt      time.Time
	RevokedAt      *time.Time
	ReplacedByID   *string
}

func sessionInfoFromRecord(rec repo.RefreshTokenRecord) SessionInfo {
	return SessionInfo{
		ID:             rec.ID,
		OrganizationID: rec.OrganizationID,
		Role:           rec.Role,
		CreatedAt:      rec.CreatedAt.UTC(),
		ExpiresAt:      rec.ExpiresAt.UTC(),
		RevokedAt:      rec.RevokedAt,
		ReplacedByID:   rec.ReplacedByID,
	}
}

// ListSessions 列出当前登录用户的全部 refresh token（含已吊销/过期的，由前端按
// revoked_at / expires_at 渲染状态）。
func (s *AuthService) ListSessions(ctx context.Context) ([]SessionInfo, error) {
	auth, ok := RequestAuthFromContext(ctx)
	if !ok || auth.UserID == "" {
		return nil, ErrUnauthorized
	}
	if s.refreshRepo == nil {
		return []SessionInfo{}, nil
	}
	records, err := s.refreshRepo.ListByUserID(ctx, auth.UserID)
	if err != nil {
		return nil, err
	}
	out := make([]SessionInfo, 0, len(records))
	for _, rec := range records {
		out = append(out, sessionInfoFromRecord(rec))
	}
	return out, nil
}

// RevokeSession 由用户主动吊销自己名下的某个 refresh token。
//
// 跨用户访问统一返回 ErrNotFound 而不是 Forbidden，避免泄露 session id 是否存在。
// 已吊销的 session 视为幂等成功。
func (s *AuthService) RevokeSession(ctx context.Context, sessionID string) error {
	auth, ok := RequestAuthFromContext(ctx)
	if !ok || auth.UserID == "" {
		return ErrUnauthorized
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session id is required: %w", domain.ErrInvalidInput)
	}
	if s.refreshRepo == nil {
		return domain.ErrNotFound
	}
	rec, err := s.refreshRepo.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if rec.UserID != auth.UserID {
		return domain.ErrNotFound
	}
	if rec.RevokedAt != nil {
		return nil
	}
	return s.refreshRepo.Revoke(ctx, sessionID, nil)
}
