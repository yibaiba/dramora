package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/service"
)

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type accountAPIKeyRequest struct {
	Name      string     `json:"name"`
	Scope     string     `json:"scope"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type toggleAccountAPIKeyRequest struct {
	IsActive bool `json:"is_active"`
}

type accountAPIKeyResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	TokenPreview string  `json:"token_preview"`
	Scope        string  `json:"scope"`
	IsActive     bool    `json:"is_active"`
	ExpiresAt    *string `json:"expires_at,omitempty"`
	LastUsedAt   *string `json:"last_used_at,omitempty"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type createdAccountAPIKeyResponse struct {
	Key   accountAPIKeyResponse `json:"key"`
	Token string                `json:"token"`
}

func accountAPIKeyDTO(item service.UserAPIKeyInfo) accountAPIKeyResponse {
	return accountAPIKeyResponse{
		ID:           item.ID,
		Name:         item.Name,
		TokenPreview: item.TokenPreview,
		Scope:        item.Scope,
		IsActive:     item.IsActive,
		ExpiresAt:    formatOptionalRFC3339(item.ExpiresAt),
		LastUsedAt:   formatOptionalRFC3339(item.LastUsedAt),
		CreatedAt:    item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (a *api) changeAccountPassword(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	var request changePasswordRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid json")
		return
	}
	if err := a.authService.ChangePassword(r.Context(), service.ChangePasswordInput{
		CurrentPassword: request.CurrentPassword,
		NewPassword:     request.NewPassword,
	}); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *api) listAccountAPIKeys(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	items, err := a.authService.ListUserAPIKeys(r.Context())
	if err != nil {
		writeAuthError(w, err)
		return
	}
	out := make([]accountAPIKeyResponse, 0, len(items))
	for _, item := range items {
		out = append(out, accountAPIKeyDTO(item))
	}
	writeJSON(w, http.StatusOK, Envelope{"api_keys": out})
}

func (a *api) createAccountAPIKey(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	var request accountAPIKeyRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid json")
		return
	}
	created, err := a.authService.CreateUserAPIKey(r.Context(), service.CreateUserAPIKeyInput{
		Name:      request.Name,
		Scope:     request.Scope,
		ExpiresAt: request.ExpiresAt,
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, Envelope{
		"api_key": createdAccountAPIKeyResponse{
			Key:   accountAPIKeyDTO(created.Key),
			Token: created.Token,
		},
	})
}

func (a *api) updateAccountAPIKey(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	var request accountAPIKeyRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid json")
		return
	}
	item, err := a.authService.UpdateUserAPIKey(r.Context(), chi.URLParam(r, "keyId"), service.UpdateUserAPIKeyInput{
		Name:      request.Name,
		Scope:     request.Scope,
		ExpiresAt: request.ExpiresAt,
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"api_key": accountAPIKeyDTO(item)})
}

func (a *api) toggleAccountAPIKey(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	var request toggleAccountAPIKeyRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid json")
		return
	}
	item, err := a.authService.ToggleUserAPIKey(r.Context(), chi.URLParam(r, "keyId"), request.IsActive)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"api_key": accountAPIKeyDTO(item)})
}

func (a *api) deleteAccountAPIKey(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}
	if err := a.authService.DeleteUserAPIKey(r.Context(), chi.URLParam(r, "keyId")); err != nil {
		writeAuthError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func formatOptionalRFC3339(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}
