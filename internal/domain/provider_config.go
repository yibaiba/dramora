package domain

import "time"

type ProviderConfig struct {
	ID             string
	Capability     string
	ProviderType   string
	BaseURL        string
	APIKey         string
	Model          string
	CreditsPerUnit int
	CreditUnit     string
	TimeoutMS      int
	MaxRetries     int
	IsEnabled      bool
	UpdatedAt      time.Time
	UpdatedBy      string
}

// ResolvedProviderType returns the explicit provider_type when set,
// otherwise falls back to "openai" so legacy rows keep working.
func (c ProviderConfig) ResolvedProviderType() string {
	if c.ProviderType == "" {
		return "openai"
	}
	return c.ProviderType
}

func (c ProviderConfig) MaskedAPIKey() string {
	if len(c.APIKey) <= 8 {
		return "****"
	}
	return c.APIKey[:4] + "****" + c.APIKey[len(c.APIKey)-4:]
}
