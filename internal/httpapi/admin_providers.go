package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/service"
)

type providerConfigDTO struct {
	ID             string `json:"id"`
	Capability     string `json:"capability"`
	ProviderType   string `json:"provider_type"`
	BaseURL        string `json:"base_url"`
	APIKey         string `json:"api_key"`
	Model          string `json:"model"`
	CreditsPerUnit int    `json:"credits_per_unit"`
	CreditUnit     string `json:"credit_unit"`
	TimeoutMS      int    `json:"timeout_ms"`
	MaxRetries     int    `json:"max_retries"`
	IsEnabled      bool   `json:"is_enabled"`
	UpdatedAt      string `json:"updated_at"`
	UpdatedBy      string `json:"updated_by"`
}

func providerConfigToDTO(c domain.ProviderConfig) providerConfigDTO {
	return providerConfigDTO{
		ID:             c.ID,
		Capability:     c.Capability,
		ProviderType:   c.ResolvedProviderType(),
		BaseURL:        c.BaseURL,
		APIKey:         c.MaskedAPIKey(),
		Model:          c.Model,
		CreditsPerUnit: c.CreditsPerUnit,
		CreditUnit:     c.CreditUnit,
		TimeoutMS:      c.TimeoutMS,
		MaxRetries:     c.MaxRetries,
		IsEnabled:      c.IsEnabled,
		UpdatedAt:      c.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedBy:      c.UpdatedBy,
	}
}

func (a *api) listProviderConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := a.providerService.ListProviderConfigs(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	dtos := make([]providerConfigDTO, len(configs))
	for i, c := range configs {
		dtos[i] = providerConfigToDTO(c)
	}
	writeJSON(w, http.StatusOK, Envelope{"providers": dtos})
}

type saveProviderConfigRequest struct {
	Capability     string `json:"capability"`
	ProviderType   string `json:"provider_type"`
	BaseURL        string `json:"base_url"`
	APIKey         string `json:"api_key"`
	Model          string `json:"model"`
	CreditsPerUnit int    `json:"credits_per_unit"`
	CreditUnit     string `json:"credit_unit"`
	TimeoutMS      int    `json:"timeout_ms"`
	MaxRetries     int    `json:"max_retries"`
}

func (a *api) saveProviderConfig(w http.ResponseWriter, r *http.Request) {
	var req saveProviderConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	cfg, err := a.providerService.SaveProviderConfig(r.Context(), service.SaveProviderConfigInput{
		Capability:     req.Capability,
		ProviderType:   req.ProviderType,
		BaseURL:        req.BaseURL,
		APIKey:         req.APIKey,
		Model:          req.Model,
		CreditsPerUnit: req.CreditsPerUnit,
		CreditUnit:     req.CreditUnit,
		TimeoutMS:      req.TimeoutMS,
		MaxRetries:     req.MaxRetries,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"provider": providerConfigToDTO(cfg)})
}

func (a *api) testProviderConfig(w http.ResponseWriter, r *http.Request) {
	capability := chi.URLParam(r, "capability")
	if capability == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "capability required")
		return
	}
	result := a.providerService.TestProviderConfig(r.Context(), capability)
	writeJSON(w, http.StatusOK, Envelope{"test_result": result})
}

func (a *api) smokeChatProvider(w http.ResponseWriter, r *http.Request) {
	result := a.providerService.SmokeChatProvider(r.Context())
	writeJSON(w, http.StatusOK, Envelope{"smoke_result": result})
}
