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
	invitations   map[string]domain.OrganizationInvitation
	tokenIndex    map[string]string // token -> invitationID
	auditEvents   []domain.InvitationAuditEvent
}

func NewMemoryIdentityRepository() *MemoryIdentityRepository {
	return &MemoryIdentityRepository{
		identities:    make(map[string]AuthIdentity),
		emailIndex:    make(map[string]string),
		organizations: make(map[string]string),
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
