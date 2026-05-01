package repo

import (
	"context"
	"sync"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
)

type MemoryRefreshTokenRepository struct {
	mu     sync.RWMutex
	byID   map[string]RefreshTokenRecord
	byHash map[string]string // hash -> id
}

func NewMemoryRefreshTokenRepository() *MemoryRefreshTokenRepository {
	return &MemoryRefreshTokenRepository{
		byID:   make(map[string]RefreshTokenRecord),
		byHash: make(map[string]string),
	}
}

func (r *MemoryRefreshTokenRepository) Create(_ context.Context, params CreateRefreshTokenParams) (RefreshTokenRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byHash[params.TokenHash]; exists {
		return RefreshTokenRecord{}, domain.ErrInvalidInput
	}
	rec := RefreshTokenRecord{
		ID:             params.ID,
		UserID:         params.UserID,
		OrganizationID: params.OrganizationID,
		Role:           params.Role,
		TokenHash:      params.TokenHash,
		CreatedAt:      time.Now().UTC(),
		ExpiresAt:      params.ExpiresAt.UTC(),
	}
	r.byID[rec.ID] = rec
	r.byHash[rec.TokenHash] = rec.ID
	return rec, nil
}

func (r *MemoryRefreshTokenRepository) GetByHash(_ context.Context, tokenHash string) (RefreshTokenRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byHash[tokenHash]
	if !ok {
		return RefreshTokenRecord{}, domain.ErrNotFound
	}
	rec, ok := r.byID[id]
	if !ok {
		return RefreshTokenRecord{}, domain.ErrNotFound
	}
	return rec, nil
}

func (r *MemoryRefreshTokenRepository) GetByID(_ context.Context, id string) (RefreshTokenRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rec, ok := r.byID[id]
	if !ok {
		return RefreshTokenRecord{}, domain.ErrNotFound
	}
	return rec, nil
}

func (r *MemoryRefreshTokenRepository) ListByUserID(_ context.Context, userID string) ([]RefreshTokenRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]RefreshTokenRecord, 0)
	for _, rec := range r.byID {
		if rec.UserID == userID {
			out = append(out, rec)
		}
	}
	// Order by CreatedAt desc for stable presentation.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].CreatedAt.After(out[j-1].CreatedAt); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out, nil
}

func (r *MemoryRefreshTokenRepository) Revoke(_ context.Context, id string, replacedByID *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	if rec.RevokedAt == nil {
		now := time.Now().UTC()
		rec.RevokedAt = &now
	}
	if replacedByID != nil {
		rec.ReplacedByID = replacedByID
	}
	r.byID[id] = rec
	return nil
}
