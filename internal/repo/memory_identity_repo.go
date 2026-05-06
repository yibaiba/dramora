package repo

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
)

type MemoryIdentityRepository struct {
	mu            sync.RWMutex
	identities    map[string]AuthIdentity
	emailIndex    map[string]string
	organizations map[string]string // orgID -> name
	userAPIKeys   map[string]map[string]domain.UserAPIKey
	invitations   map[string]domain.OrganizationInvitation
	tokenIndex    map[string]string // token -> invitationID
	auditEvents   []domain.InvitationAuditEvent
}

func NewMemoryIdentityRepository() *MemoryIdentityRepository {
	return &MemoryIdentityRepository{
		identities:    make(map[string]AuthIdentity),
		emailIndex:    make(map[string]string),
		organizations: make(map[string]string),
		userAPIKeys:   make(map[string]map[string]domain.UserAPIKey),
		invitations:   make(map[string]domain.OrganizationInvitation),
		tokenIndex:    make(map[string]string),
	}
}

func (r *MemoryIdentityRepository) CreateUserWithMembership(
	_ context.Context,
	params CreateUserWithMembershipParams,
) (AuthIdentity, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	emailKey := strings.ToLower(strings.TrimSpace(params.Email))
	if _, exists := r.emailIndex[emailKey]; exists {
		return AuthIdentity{}, domain.ErrInvalidInput
	}

	now := time.Now().UTC()
	identity := AuthIdentity{
		User: domain.User{
			ID:          params.UserID,
			Email:       params.Email,
			DisplayName: params.DisplayName,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		OrganizationID: params.OrganizationID,
		Role:           params.Role,
		PasswordHash:   params.PasswordHash,
	}
	r.identities[identity.User.ID] = identity
	r.emailIndex[emailKey] = identity.User.ID
	return identity, nil
}

func (r *MemoryIdentityRepository) GetAuthIdentityByEmail(_ context.Context, email string) (AuthIdentity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userID, ok := r.emailIndex[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return AuthIdentity{}, domain.ErrNotFound
	}
	return r.identities[userID], nil
}

func (r *MemoryIdentityRepository) GetAuthIdentityByUserID(_ context.Context, userID string) (AuthIdentity, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	identity, ok := r.identities[userID]
	if !ok {
		return AuthIdentity{}, domain.ErrNotFound
	}
	return identity, nil
}

func (r *MemoryIdentityRepository) UpdateUserPasswordHash(
	_ context.Context,
	userID, passwordHash string,
	updatedAt time.Time,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	identity, ok := r.identities[userID]
	if !ok {
		return domain.ErrNotFound
	}
	identity.PasswordHash = passwordHash
	identity.User.UpdatedAt = updatedAt.UTC()
	r.identities[userID] = identity
	return nil
}

func (r *MemoryIdentityRepository) ListUserAPIKeys(_ context.Context, userID string) ([]domain.UserAPIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := r.userAPIKeys[userID]
	out := make([]domain.UserAPIKey, 0, len(keys))
	for _, item := range keys {
		out = append(out, cloneUserAPIKey(item))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (r *MemoryIdentityRepository) CreateUserAPIKey(
	_ context.Context,
	params CreateUserAPIKeyParams,
) (domain.UserAPIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.identities[params.UserID]; !ok {
		return domain.UserAPIKey{}, domain.ErrNotFound
	}
	if _, ok := r.userAPIKeys[params.UserID]; !ok {
		r.userAPIKeys[params.UserID] = make(map[string]domain.UserAPIKey)
	}
	if _, exists := r.userAPIKeys[params.UserID][params.KeyID]; exists {
		return domain.UserAPIKey{}, domain.ErrInvalidInput
	}
	item := domain.UserAPIKey{
		ID:           params.KeyID,
		UserID:       params.UserID,
		Name:         params.Name,
		TokenPreview: params.TokenPreview,
		Scope:        params.Scope,
		IsActive:     params.IsActive,
		ExpiresAt:    cloneTimePtr(params.ExpiresAt),
		CreatedAt:    params.CreatedAt.UTC(),
		UpdatedAt:    params.UpdatedAt.UTC(),
	}
	r.userAPIKeys[params.UserID][params.KeyID] = item
	return cloneUserAPIKey(item), nil
}

func (r *MemoryIdentityRepository) UpdateUserAPIKey(
	_ context.Context,
	params UpdateUserAPIKeyParams,
) (domain.UserAPIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	keys, ok := r.userAPIKeys[params.UserID]
	if !ok {
		return domain.UserAPIKey{}, domain.ErrNotFound
	}
	item, ok := keys[params.KeyID]
	if !ok {
		return domain.UserAPIKey{}, domain.ErrNotFound
	}
	item.Name = params.Name
	item.Scope = params.Scope
	item.ExpiresAt = cloneTimePtr(params.ExpiresAt)
	item.UpdatedAt = params.UpdatedAt.UTC()
	keys[params.KeyID] = item
	return cloneUserAPIKey(item), nil
}

func (r *MemoryIdentityRepository) SetUserAPIKeyActive(
	_ context.Context,
	userID, keyID string,
	isActive bool,
	updatedAt time.Time,
) (domain.UserAPIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	keys, ok := r.userAPIKeys[userID]
	if !ok {
		return domain.UserAPIKey{}, domain.ErrNotFound
	}
	item, ok := keys[keyID]
	if !ok {
		return domain.UserAPIKey{}, domain.ErrNotFound
	}
	item.IsActive = isActive
	item.UpdatedAt = updatedAt.UTC()
	keys[keyID] = item
	return cloneUserAPIKey(item), nil
}

func (r *MemoryIdentityRepository) DeleteUserAPIKey(_ context.Context, userID, keyID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	keys, ok := r.userAPIKeys[userID]
	if !ok {
		return domain.ErrNotFound
	}
	if _, exists := keys[keyID]; !exists {
		return domain.ErrNotFound
	}
	delete(keys, keyID)
	return nil
}

func (r *MemoryIdentityRepository) ListOrganizationMembers(
	_ context.Context,
	organizationID string,
) ([]domain.OrganizationMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.OrganizationMember, 0)
	for _, identity := range r.identities {
		if identity.OrganizationID != organizationID {
			continue
		}
		out = append(out, domain.OrganizationMember{
			OrganizationID: identity.OrganizationID,
			UserID:         identity.User.ID,
			Email:          identity.User.Email,
			DisplayName:    identity.User.DisplayName,
			Role:           identity.Role,
			JoinedAt:       identity.User.CreatedAt.UTC(),
			LastActivityAt: identity.User.UpdatedAt.UTC(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		left := organizationRoleRank(out[i].Role)
		right := organizationRoleRank(out[j].Role)
		if left != right {
			return left < right
		}
		return strings.ToLower(out[i].Email) < strings.ToLower(out[j].Email)
	})
	return out, nil
}

func (r *MemoryIdentityRepository) GetOrganizationMember(
	_ context.Context,
	organizationID, userID string,
) (domain.OrganizationMember, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	identity, ok := r.identities[userID]
	if !ok || identity.OrganizationID != organizationID {
		return domain.OrganizationMember{}, domain.ErrNotFound
	}
	return domain.OrganizationMember{
		OrganizationID: identity.OrganizationID,
		UserID:         identity.User.ID,
		Email:          identity.User.Email,
		DisplayName:    identity.User.DisplayName,
		Role:           identity.Role,
		JoinedAt:       identity.User.CreatedAt.UTC(),
		LastActivityAt: identity.User.UpdatedAt.UTC(),
	}, nil
}

func (r *MemoryIdentityRepository) UpdateOrganizationMemberRole(
	_ context.Context,
	organizationID, userID, role string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	identity, ok := r.identities[userID]
	if !ok || identity.OrganizationID != organizationID {
		return domain.ErrNotFound
	}
	identity.Role = role
	r.identities[userID] = identity
	return nil
}

func (r *MemoryIdentityRepository) RemoveOrganizationMember(
	_ context.Context,
	organizationID, userID string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	identity, ok := r.identities[userID]
	if !ok || identity.OrganizationID != organizationID {
		return domain.ErrNotFound
	}
	delete(r.identities, userID)
	delete(r.emailIndex, strings.ToLower(strings.TrimSpace(identity.User.Email)))
	return nil
}

func (r *MemoryIdentityRepository) CreateOrganization(_ context.Context, params CreateOrganizationParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.organizations[params.OrganizationID]; exists {
		return domain.ErrInvalidInput
	}
	r.organizations[params.OrganizationID] = params.Name
	return nil
}

func (r *MemoryIdentityRepository) CreateInvitation(_ context.Context, params CreateInvitationParams) (domain.OrganizationInvitation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tokenIndex[params.Token]; exists {
		return domain.OrganizationInvitation{}, domain.ErrInvalidInput
	}
	now := time.Now().UTC()
	inv := domain.OrganizationInvitation{
		ID:              params.InvitationID,
		OrganizationID:  params.OrganizationID,
		Email:           strings.ToLower(strings.TrimSpace(params.Email)),
		Role:            params.Role,
		Token:           params.Token,
		InvitedByUserID: params.InvitedByUserID,
		Status:          domain.InvitationStatusPending,
		ExpiresAt:       params.ExpiresAt.UTC(),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	r.invitations[inv.ID] = inv
	r.tokenIndex[inv.Token] = inv.ID
	return inv, nil
}

func (r *MemoryIdentityRepository) GetInvitationByToken(_ context.Context, token string) (domain.OrganizationInvitation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.tokenIndex[token]
	if !ok {
		return domain.OrganizationInvitation{}, domain.ErrNotFound
	}
	return r.invitations[id], nil
}

func (r *MemoryIdentityRepository) MarkInvitationAccepted(_ context.Context, invitationID, userID string, acceptedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.invitations[invitationID]
	if !ok || inv.Status != domain.InvitationStatusPending {
		return domain.ErrNotFound
	}
	at := acceptedAt.UTC()
	inv.Status = domain.InvitationStatusAccepted
	inv.AcceptedAt = &at
	inv.AcceptedByUserID = userID
	inv.UpdatedAt = at
	r.invitations[invitationID] = inv
	return nil
}

func (r *MemoryIdentityRepository) ListOrganizationInvitations(_ context.Context, organizationID string) ([]domain.OrganizationInvitation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []domain.OrganizationInvitation
	for _, inv := range r.invitations {
		if inv.OrganizationID == organizationID {
			out = append(out, inv)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (r *MemoryIdentityRepository) RevokeInvitation(_ context.Context, invitationID, organizationID string, revokedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.invitations[invitationID]
	if !ok || inv.OrganizationID != organizationID || inv.Status != domain.InvitationStatusPending {
		return domain.ErrNotFound
	}
	inv.Status = domain.InvitationStatusRevoked
	inv.UpdatedAt = revokedAt.UTC()
	r.invitations[invitationID] = inv
	return nil
}

func (r *MemoryIdentityRepository) AppendInvitationAuditEvent(
	_ context.Context,
	params AppendInvitationAuditParams,
) (domain.InvitationAuditEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ev := domain.InvitationAuditEvent{
		ID:             params.EventID,
		OrganizationID: params.OrganizationID,
		InvitationID:   params.InvitationID,
		Action:         params.Action,
		ActorUserID:    params.ActorUserID,
		ActorEmail:     params.ActorEmail,
		Email:          params.Email,
		Role:           params.Role,
		Note:           params.Note,
		CreatedAt:      params.CreatedAt.UTC(),
	}
	r.auditEvents = append(r.auditEvents, ev)
	return ev, nil
}

func (r *MemoryIdentityRepository) ListInvitationAuditEvents(
	_ context.Context,
	filter InvitationAuditFilter,
) (InvitationAuditPage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	actionSet := map[string]struct{}{}
	for _, a := range filter.Actions {
		actionSet[a] = struct{}{}
	}
	emailNeedle := strings.ToLower(strings.TrimSpace(filter.Email))
	matched := make([]domain.InvitationAuditEvent, 0)
	for i := len(r.auditEvents) - 1; i >= 0; i-- {
		ev := r.auditEvents[i]
		if ev.OrganizationID != filter.OrganizationID {
			continue
		}
		if len(actionSet) > 0 {
			if _, ok := actionSet[ev.Action]; !ok {
				continue
			}
		}
		if emailNeedle != "" && !strings.Contains(strings.ToLower(ev.Email), emailNeedle) {
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
	if offset >= len(matched) {
		return InvitationAuditPage{Events: []domain.InvitationAuditEvent{}, HasMore: false}, nil
	}
	end := offset + limit
	hasMore := false
	if end < len(matched) {
		hasMore = true
	} else {
		end = len(matched)
	}
	return InvitationAuditPage{Events: append([]domain.InvitationAuditEvent(nil), matched[offset:end]...), HasMore: hasMore}, nil
}

func organizationRoleRank(role string) int {
	switch role {
	case "owner":
		return 0
	case "admin":
		return 1
	case "editor":
		return 2
	default:
		return 3
	}
}

func cloneUserAPIKey(item domain.UserAPIKey) domain.UserAPIKey {
	item.ExpiresAt = cloneTimePtr(item.ExpiresAt)
	item.LastUsedAt = cloneTimePtr(item.LastUsedAt)
	return item
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}
