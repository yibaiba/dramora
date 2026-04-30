package repo

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
)

type MemoryIdentityRepository struct {
	mu         sync.RWMutex
	identities map[string]AuthIdentity
	emailIndex map[string]string
}

func NewMemoryIdentityRepository() *MemoryIdentityRepository {
	return &MemoryIdentityRepository{
		identities: make(map[string]AuthIdentity),
		emailIndex: make(map[string]string),
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
