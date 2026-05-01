package repo

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LLMTelemetryAggregateScope identifies which axis the aggregate row groups by.
type LLMTelemetryAggregateScope string

const (
	LLMTelemetryAggregateScopeVendor     LLMTelemetryAggregateScope = "vendor"
	LLMTelemetryAggregateScopeCapability LLMTelemetryAggregateScope = "capability"
)

// LLMTelemetryAggregateRow describes a single (scope, key) counter row.
type LLMTelemetryAggregateRow struct {
	Scope           LLMTelemetryAggregateScope
	Key             string
	Counter         uint64
	ErrorCounter    uint64
	TotalDurationMS int64
}

// LLMTelemetryRepository persists per-vendor / per-capability counters so
// LLM telemetry survives process restarts (mirrors WorkerMetricsRepository).
type LLMTelemetryRepository interface {
	LoadAll(ctx context.Context) ([]LLMTelemetryAggregateRow, error)
	RecordCall(ctx context.Context, scope LLMTelemetryAggregateScope, key string, durationMS int64, success bool) error
}

// MemoryLLMTelemetryRepository provides an in-memory implementation, used by tests.
type MemoryLLMTelemetryRepository struct {
	mu   sync.Mutex
	rows map[string]LLMTelemetryAggregateRow
}

func NewMemoryLLMTelemetryRepository() *MemoryLLMTelemetryRepository {
	return &MemoryLLMTelemetryRepository{rows: map[string]LLMTelemetryAggregateRow{}}
}

func memTelemetryKey(scope LLMTelemetryAggregateScope, key string) string {
	return string(scope) + "|" + key
}

func (r *MemoryLLMTelemetryRepository) LoadAll(_ context.Context) ([]LLMTelemetryAggregateRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]LLMTelemetryAggregateRow, 0, len(r.rows))
	for _, row := range r.rows {
		out = append(out, row)
	}
	return out, nil
}

func (r *MemoryLLMTelemetryRepository) RecordCall(
	_ context.Context,
	scope LLMTelemetryAggregateScope,
	key string,
	durationMS int64,
	success bool,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := memTelemetryKey(scope, key)
	row := r.rows[k]
	row.Scope = scope
	row.Key = key
	row.Counter++
	row.TotalDurationMS += durationMS
	if !success {
		row.ErrorCounter++
	}
	r.rows[k] = row
	return nil
}

// SQLiteLLMTelemetryRepository persists aggregates in SQLite.
type SQLiteLLMTelemetryRepository struct {
	db *sql.DB
}

func NewSQLiteLLMTelemetryRepository(db *sql.DB) *SQLiteLLMTelemetryRepository {
	return &SQLiteLLMTelemetryRepository{db: db}
}

func (r *SQLiteLLMTelemetryRepository) LoadAll(ctx context.Context) ([]LLMTelemetryAggregateRow, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT scope, key, counter, error_counter, total_duration_ms FROM llm_telemetry_aggregate`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LLMTelemetryAggregateRow, 0)
	for rows.Next() {
		var scope, key string
		var counter, errCounter, totalDur int64
		if err := rows.Scan(&scope, &key, &counter, &errCounter, &totalDur); err != nil {
			return nil, err
		}
		out = append(out, LLMTelemetryAggregateRow{
			Scope:           LLMTelemetryAggregateScope(scope),
			Key:             key,
			Counter:         uint64(counter),
			ErrorCounter:    uint64(errCounter),
			TotalDurationMS: totalDur,
		})
	}
	return out, rows.Err()
}

func (r *SQLiteLLMTelemetryRepository) RecordCall(
	ctx context.Context,
	scope LLMTelemetryAggregateScope,
	key string,
	durationMS int64,
	success bool,
) error {
	errInc := int64(0)
	if !success {
		errInc = 1
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO llm_telemetry_aggregate (scope, key, counter, error_counter, total_duration_ms, updated_at)
		VALUES (?, ?, 1, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		ON CONFLICT(scope, key) DO UPDATE SET
			counter = llm_telemetry_aggregate.counter + 1,
			error_counter = llm_telemetry_aggregate.error_counter + excluded.error_counter,
			total_duration_ms = llm_telemetry_aggregate.total_duration_ms + excluded.total_duration_ms,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
	`, string(scope), key, errInc, durationMS)
	return err
}

// PostgresLLMTelemetryRepository persists aggregates in Postgres.
type PostgresLLMTelemetryRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresLLMTelemetryRepository(pool *pgxpool.Pool) *PostgresLLMTelemetryRepository {
	return &PostgresLLMTelemetryRepository{pool: pool}
}

func (r *PostgresLLMTelemetryRepository) LoadAll(ctx context.Context) ([]LLMTelemetryAggregateRow, error) {
	rows, err := r.pool.Query(ctx, `SELECT scope, key, counter, error_counter, total_duration_ms FROM llm_telemetry_aggregate`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LLMTelemetryAggregateRow, 0)
	for rows.Next() {
		var scope, key string
		var counter, errCounter, totalDur int64
		if err := rows.Scan(&scope, &key, &counter, &errCounter, &totalDur); err != nil {
			return nil, err
		}
		out = append(out, LLMTelemetryAggregateRow{
			Scope:           LLMTelemetryAggregateScope(scope),
			Key:             key,
			Counter:         uint64(counter),
			ErrorCounter:    uint64(errCounter),
			TotalDurationMS: totalDur,
		})
	}
	if err := rows.Err(); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return out, nil
}

func (r *PostgresLLMTelemetryRepository) RecordCall(
	ctx context.Context,
	scope LLMTelemetryAggregateScope,
	key string,
	durationMS int64,
	success bool,
) error {
	errInc := int64(0)
	if !success {
		errInc = 1
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO llm_telemetry_aggregate (scope, key, counter, error_counter, total_duration_ms, updated_at)
		VALUES ($1, $2, 1, $3, $4, NOW())
		ON CONFLICT (scope, key) DO UPDATE SET
			counter = llm_telemetry_aggregate.counter + 1,
			error_counter = llm_telemetry_aggregate.error_counter + EXCLUDED.error_counter,
			total_duration_ms = llm_telemetry_aggregate.total_duration_ms + EXCLUDED.total_duration_ms,
			updated_at = NOW()
	`, string(scope), key, errInc, durationMS)
	return err
}
