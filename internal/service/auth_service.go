package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
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

const defaultAuthTokenTTL = 7 * 24 * time.Hour

type AuthSession struct {
	Token          string
	User           domain.User
	OrganizationID string
	Role           string
	ExpiresAt      time.Time
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
	identityRepo          repo.IdentityRepository
	defaultOrganizationID string
	jwtSecret             []byte
	tokenTTL              time.Duration
}

func NewAuthService(identityRepo repo.IdentityRepository, defaultOrganizationID string, jwtSecret string) *AuthService {
	return &AuthService{
		identityRepo:          identityRepo,
		defaultOrganizationID: defaultOrganizationID,
		jwtSecret:             []byte(jwtSecret),
		tokenTTL:              defaultAuthTokenTTL,
	}
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
	}

	return s.buildSession(identity)
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

	return s.buildSession(identity)
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

func (s *AuthService) buildSession(identity repo.AuthIdentity) (AuthSession, error) {
	expiresAt := time.Now().UTC().Add(s.tokenTTL)
	token, err := s.signToken(authClaims{
		Subject:        identity.User.ID,
		OrganizationID: identity.OrganizationID,
		Role:           identity.Role,
		ExpiresAt:      expiresAt.Unix(),
	})
	if err != nil {
		return AuthSession{}, err
	}
	return AuthSession{
		Token:          token,
		User:           identity.User,
		OrganizationID: identity.OrganizationID,
		Role:           identity.Role,
		ExpiresAt:      expiresAt,
	}, nil
}

// IssueSessionForIdentity signs a session token for an already-resolved
// identity. Useful for seeding tests or service-to-service flows where
// password validation has already happened (or doesn't apply).
func (s *AuthService) IssueSessionForIdentity(identity repo.AuthIdentity) (AuthSession, error) {
	return s.buildSession(identity)
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

	return s.identityRepo.CreateInvitation(ctx, repo.CreateInvitationParams{
		InvitationID:    uuid.NewString(),
		OrganizationID:  auth.OrganizationID,
		Email:           email,
		Role:            role,
		Token:           token,
		InvitedByUserID: auth.UserID,
		ExpiresAt:       time.Now().UTC().Add(defaultInvitationTTL),
	})
}

func (s *AuthService) ListInvitations(ctx context.Context) ([]domain.OrganizationInvitation, error) {
	auth, ok := RequestAuthFromContext(ctx)
	if !ok || auth.OrganizationID == "" {
		return nil, ErrUnauthorized
	}
	return s.identityRepo.ListOrganizationInvitations(ctx, auth.OrganizationID)
}
