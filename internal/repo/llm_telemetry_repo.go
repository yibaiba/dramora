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
	Reset(ctx context.Context) error

	// Daily bucket APIs feed the rolling N-day window.
	RecordDaily(ctx context.Context, scope LLMTelemetryAggregateScope, key string, dayUTC string, durationMS int64, success bool) error
	LoadDailySince(ctx context.Context, sinceDayUTC string) ([]LLMTelemetryDailyRow, error)
}

// LLMTelemetryDailyRow represents a (scope, key, day_utc) bucket counter.
type LLMTelemetryDailyRow struct {
	Scope           LLMTelemetryAggregateScope
	Key             string
	DayUTC          string // YYYY-MM-DD
	Counter         uint64
	ErrorCounter    uint64
	TotalDurationMS int64
}

// MemoryLLMTelemetryRepository provides an in-memory implementation, used by tests.
type MemoryLLMTelemetryRepository struct {
	mu    sync.Mutex
	rows  map[string]LLMTelemetryAggregateRow
	daily map[string]LLMTelemetryDailyRow
}

func NewMemoryLLMTelemetryRepository() *MemoryLLMTelemetryRepository {
	return &MemoryLLMTelemetryRepository{
		rows:  map[string]LLMTelemetryAggregateRow{},
		daily: map[string]LLMTelemetryDailyRow{},
	}
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

func (r *MemoryLLMTelemetryRepository) Reset(_ context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = map[string]LLMTelemetryAggregateRow{}
	r.daily = map[string]LLMTelemetryDailyRow{}
	return nil
}

func memTelemetryDailyKey(scope LLMTelemetryAggregateScope, key, day string) string {
	return string(scope) + "|" + key + "|" + day
}

func (r *MemoryLLMTelemetryRepository) RecordDaily(
	_ context.Context,
	scope LLMTelemetryAggregateScope,
	key string,
	dayUTC string,
	durationMS int64,
	success bool,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := memTelemetryDailyKey(scope, key, dayUTC)
	row := r.daily[k]
	row.Scope = scope
	row.Key = key
	row.DayUTC = dayUTC
	row.Counter++
	row.TotalDurationMS += durationMS
	if !success {
		row.ErrorCounter++
	}
	r.daily[k] = row
	return nil
}

func (r *MemoryLLMTelemetryRepository) LoadDailySince(
	_ context.Context,
	sinceDayUTC string,
) ([]LLMTelemetryDailyRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]LLMTelemetryDailyRow, 0, len(r.daily))
	for _, row := range r.daily {
		if row.DayUTC >= sinceDayUTC {
			out = append(out, row)
		}
	}
	return out, nil
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

func (r *PostgresLLMTelemetryRepository) Reset(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, `DELETE FROM llm_telemetry_aggregate`); err != nil {
		return err
	}
	_, err := r.pool.Exec(ctx, `DELETE FROM llm_telemetry_daily`)
	return err
}

// --- Daily bucket implementations ---

func (r *SQLiteLLMTelemetryRepository) Reset(ctx context.Context) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM llm_telemetry_aggregate`); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM llm_telemetry_daily`)
	return err
}

func (r *SQLiteLLMTelemetryRepository) RecordDaily(
	ctx context.Context,
	scope LLMTelemetryAggregateScope,
	key string,
	dayUTC string,
	durationMS int64,
	success bool,
) error {
	errInc := int64(0)
	if !success {
		errInc = 1
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO llm_telemetry_daily (scope, key, day_utc, counter, error_counter, total_duration_ms, updated_at)
		VALUES (?, ?, ?, 1, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		ON CONFLICT(scope, key, day_utc) DO UPDATE SET
			counter = llm_telemetry_daily.counter + 1,
			error_counter = llm_telemetry_daily.error_counter + excluded.error_counter,
			total_duration_ms = llm_telemetry_daily.total_duration_ms + excluded.total_duration_ms,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
	`, string(scope), key, dayUTC, errInc, durationMS)
	return err
}

func (r *SQLiteLLMTelemetryRepository) LoadDailySince(
	ctx context.Context,
	sinceDayUTC string,
) ([]LLMTelemetryDailyRow, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT scope, key, day_utc, counter, error_counter, total_duration_ms FROM llm_telemetry_daily WHERE day_utc >= ?`,
		sinceDayUTC,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LLMTelemetryDailyRow, 0)
	for rows.Next() {
		var scope, key, day string
		var counter, errCounter, totalDur int64
		if err := rows.Scan(&scope, &key, &day, &counter, &errCounter, &totalDur); err != nil {
			return nil, err
		}
		out = append(out, LLMTelemetryDailyRow{
			Scope:           LLMTelemetryAggregateScope(scope),
			Key:             key,
			DayUTC:          day,
			Counter:         uint64(counter),
			ErrorCounter:    uint64(errCounter),
			TotalDurationMS: totalDur,
		})
	}
	return out, rows.Err()
}

func (r *PostgresLLMTelemetryRepository) RecordDaily(
	ctx context.Context,
	scope LLMTelemetryAggregateScope,
	key string,
	dayUTC string,
	durationMS int64,
	success bool,
) error {
	errInc := int64(0)
	if !success {
		errInc = 1
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO llm_telemetry_daily (scope, key, day_utc, counter, error_counter, total_duration_ms, updated_at)
		VALUES ($1, $2, $3::date, 1, $4, $5, NOW())
		ON CONFLICT (scope, key, day_utc) DO UPDATE SET
			counter = llm_telemetry_daily.counter + 1,
			error_counter = llm_telemetry_daily.error_counter + EXCLUDED.error_counter,
			total_duration_ms = llm_telemetry_daily.total_duration_ms + EXCLUDED.total_duration_ms,
			updated_at = NOW()
	`, string(scope), key, dayUTC, errInc, durationMS)
	return err
}

func (r *PostgresLLMTelemetryRepository) LoadDailySince(
	ctx context.Context,
	sinceDayUTC string,
) ([]LLMTelemetryDailyRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT scope, key, to_char(day_utc, 'YYYY-MM-DD'), counter, error_counter, total_duration_ms
		 FROM llm_telemetry_daily WHERE day_utc >= $1::date`,
		sinceDayUTC,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LLMTelemetryDailyRow, 0)
	for rows.Next() {
		var scope, key, day string
		var counter, errCounter, totalDur int64
		if err := rows.Scan(&scope, &key, &day, &counter, &errCounter, &totalDur); err != nil {
			return nil, err
		}
		out = append(out, LLMTelemetryDailyRow{
			Scope:           LLMTelemetryAggregateScope(scope),
			Key:             key,
			DayUTC:          day,
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
