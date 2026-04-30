package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/service"
)

type authRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

type authSessionResponse struct {
	Token          string       `json:"token"`
	User           userResponse `json:"user"`
	OrganizationID string       `json:"organization_id"`
	Role           string       `json:"role"`
	ExpiresAt      time.Time    `json:"expires_at"`
}

type userResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

func userDTO(user domain.User) userResponse {
	return userResponse{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
	}
}

func authSessionDTO(session service.AuthSession) authSessionResponse {
	return authSessionResponse{
		Token:          session.Token,
		User:           userDTO(session.User),
		OrganizationID: session.OrganizationID,
		Role:           session.Role,
		ExpiresAt:      session.ExpiresAt.UTC(),
	}
}

func (a *api) register(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}

	var request authRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	session, err := a.authService.Register(r.Context(), service.RegisterInput{
		Email:       request.Email,
		DisplayName: request.DisplayName,
		Password:    request.Password,
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]authSessionResponse{
		"session": authSessionDTO(session),
	})
}

func (a *api) login(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}

	var request authRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	session, err := a.authService.Login(r.Context(), service.LoginInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]authSessionResponse{
		"session": authSessionDTO(session),
	})
}

func (a *api) currentSession(w http.ResponseWriter, r *http.Request) {
	if a.authService == nil {
		writeError(w, http.StatusNotImplemented, "not_supported", "auth service is not configured")
		return
	}

	session, err := a.authService.CurrentSession(r.Context(), r.Header.Get("Authorization"))
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]authSessionResponse{
		"session": authSessionDTO(session),
	})
}

func writeAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired credentials")
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		writeServiceError(w, err)
	}
}
