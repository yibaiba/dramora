package domain

import "time"

type ProviderConfig struct {
	ID             string
	Capability     string
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

func (c ProviderConfig) MaskedAPIKey() string {
	if len(c.APIKey) <= 8 {
		return "****"
	}
	return c.APIKey[:4] + "****" + c.APIKey[len(c.APIKey)-4:]
}
