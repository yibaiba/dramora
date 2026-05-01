package repo

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/yibaiba/dramora/internal/domain"
)

type SaveProviderConfigParams struct {
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
	UpdatedBy      string
}

type ProviderConfigRepository interface {
	ListProviderConfigs(ctx context.Context) ([]domain.ProviderConfig, error)
	GetProviderConfig(ctx context.Context, capability string) (domain.ProviderConfig, error)
	SaveProviderConfig(ctx context.Context, params SaveProviderConfigParams) (domain.ProviderConfig, error)
}

type SQLiteProviderConfigRepository struct {
	db *sql.DB
}

// MemoryProviderConfigRepository 是只用于测试与内存模式的实现。
// 当前 Container 仅在 SQLite 路径上启用 ProviderService，但 HTTP 测试需要一个
// 不依赖 SQLite 的轻量实现来覆盖 admin/providers 与 audit log 路径。
type MemoryProviderConfigRepository struct {
	mu      sync.Mutex
	byCapID map[string]domain.ProviderConfig
}

func NewMemoryProviderConfigRepository() *MemoryProviderConfigRepository {
	return &MemoryProviderConfigRepository{byCapID: map[string]domain.ProviderConfig{}}
}

func (r *MemoryProviderConfigRepository) ListProviderConfigs(_ context.Context) ([]domain.ProviderConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.ProviderConfig, 0, len(r.byCapID))
	for _, c := range r.byCapID {
		out = append(out, c)
	}
	return out, nil
}

func (r *MemoryProviderConfigRepository) GetProviderConfig(_ context.Context, capability string) (domain.ProviderConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cfg, ok := r.byCapID[capability]
	if !ok {
		return domain.ProviderConfig{}, sql.ErrNoRows
	}
	return cfg, nil
}

func (r *MemoryProviderConfigRepository) SaveProviderConfig(_ context.Context, params SaveProviderConfigParams) (domain.ProviderConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cfg := domain.ProviderConfig{
		ID:             params.ID,
		Capability:     params.Capability,
		ProviderType:   params.ProviderType,
		BaseURL:        params.BaseURL,
		APIKey:         params.APIKey,
		Model:          params.Model,
		CreditsPerUnit: params.CreditsPerUnit,
		CreditUnit:     params.CreditUnit,
		TimeoutMS:      params.TimeoutMS,
		MaxRetries:     params.MaxRetries,
		IsEnabled:      true,
		UpdatedAt:      time.Now().UTC(),
		UpdatedBy:      params.UpdatedBy,
	}
	r.byCapID[params.Capability] = cfg
	return cfg, nil
}

func NewSQLiteProviderConfigRepository(db *sql.DB) *SQLiteProviderConfigRepository {
	return &SQLiteProviderConfigRepository{db: db}
}

func (r *SQLiteProviderConfigRepository) ListProviderConfigs(ctx context.Context) ([]domain.ProviderConfig, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, capability, COALESCE(provider_type, 'openai'),
		       base_url, api_key, model, credits_per_unit, credit_unit,
		       timeout_ms, max_retries, is_enabled,
		       COALESCE(updated_at, '0001-01-01T00:00:00Z'), COALESCE(updated_by, '')
		FROM provider_configs
		ORDER BY capability`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := make([]domain.ProviderConfig, 0)
	for rows.Next() {
		c, err := scanProviderConfig(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

func (r *SQLiteProviderConfigRepository) GetProviderConfig(ctx context.Context, capability string) (domain.ProviderConfig, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, capability, COALESCE(provider_type, 'openai'),
		       base_url, api_key, model, credits_per_unit, credit_unit,
		       timeout_ms, max_retries, is_enabled,
		       COALESCE(updated_at, '0001-01-01T00:00:00Z'), COALESCE(updated_by, '')
		FROM provider_configs
		WHERE capability = ?`, capability)
	c, err := scanProviderConfig(row)
	if err == sql.ErrNoRows {
		return domain.ProviderConfig{}, domain.ErrNotFound
	}
	return c, err
}

func (r *SQLiteProviderConfigRepository) SaveProviderConfig(ctx context.Context, params SaveProviderConfigParams) (domain.ProviderConfig, error) {
	providerType := params.ProviderType
	if providerType == "" {
		providerType = "openai"
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO provider_configs (id, capability, provider_type, base_url, api_key, model, credits_per_unit, credit_unit, timeout_ms, max_retries, is_enabled, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?)
		ON CONFLICT (capability) DO UPDATE
		SET provider_type = excluded.provider_type,
		    base_url = excluded.base_url,
		    api_key = excluded.api_key,
		    model = excluded.model,
		    credits_per_unit = excluded.credits_per_unit,
		    credit_unit = excluded.credit_unit,
		    timeout_ms = excluded.timeout_ms,
		    max_retries = excluded.max_retries,
		    is_enabled = 1,
		    updated_by = excluded.updated_by,
		    updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')`,
		params.ID, params.Capability, providerType, params.BaseURL, params.APIKey, params.Model,
		params.CreditsPerUnit, params.CreditUnit, params.TimeoutMS, params.MaxRetries, params.UpdatedBy,
	)
	if err != nil {
		return domain.ProviderConfig{}, err
	}
	return r.GetProviderConfig(ctx, params.Capability)
}

func scanProviderConfig(row rowScanner) (domain.ProviderConfig, error) {
	var c domain.ProviderConfig
	err := row.Scan(
		&c.ID, &c.Capability, &c.ProviderType, &c.BaseURL, &c.APIKey, &c.Model,
		&c.CreditsPerUnit, &c.CreditUnit, &c.TimeoutMS, &c.MaxRetries,
		&c.IsEnabled, &c.UpdatedAt, &c.UpdatedBy,
	)
	if err == nil && c.ProviderType == "" {
		c.ProviderType = "openai"
	}
	return c, err
}
