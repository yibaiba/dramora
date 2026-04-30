package repo

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkerMetricKind 标识一种 worker 可观测计数器的种类。
type WorkerMetricKind string

const (
	WorkerMetricKindGenerationSkip WorkerMetricKind = "generation_skip"
	WorkerMetricKindExportSkip     WorkerMetricKind = "export_skip"
)

// WorkerMetricRow 描述单条 metric 状态行（一种类型一行）。
type WorkerMetricRow struct {
	Kind       WorkerMetricKind
	Counter    uint64
	LastReason string
	LastAt     time.Time
}

// WorkerMetricsRepository 提供 worker 可观测计数器的持久化能力。
// 计数器在进程启动时被加载到内存 atomic，运行期每次 skip 都会同步写回。
type WorkerMetricsRepository interface {
	LoadAll(ctx context.Context) ([]WorkerMetricRow, error)
	IncrementSkip(ctx context.Context, kind WorkerMetricKind, reason string, at time.Time) error
}

// MemoryWorkerMetricsRepository 提供内存实现，主要用于测试。
type MemoryWorkerMetricsRepository struct {
	mu   sync.Mutex
	rows map[WorkerMetricKind]WorkerMetricRow
}

func NewMemoryWorkerMetricsRepository() *MemoryWorkerMetricsRepository {
	return &MemoryWorkerMetricsRepository{rows: map[WorkerMetricKind]WorkerMetricRow{}}
}

func (r *MemoryWorkerMetricsRepository) LoadAll(_ context.Context) ([]WorkerMetricRow, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]WorkerMetricRow, 0, len(r.rows))
	for _, row := range r.rows {
		out = append(out, row)
	}
	return out, nil
}

func (r *MemoryWorkerMetricsRepository) IncrementSkip(
	_ context.Context,
	kind WorkerMetricKind,
	reason string,
	at time.Time,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row := r.rows[kind]
	row.Kind = kind
	row.Counter++
	row.LastReason = reason
	row.LastAt = at
	r.rows[kind] = row
	return nil
}

// SQLiteWorkerMetricsRepository 提供 SQLite 后端的持久化实现。
type SQLiteWorkerMetricsRepository struct {
	db *sql.DB
}

func NewSQLiteWorkerMetricsRepository(db *sql.DB) *SQLiteWorkerMetricsRepository {
	return &SQLiteWorkerMetricsRepository{db: db}
}

func (r *SQLiteWorkerMetricsRepository) LoadAll(ctx context.Context) ([]WorkerMetricRow, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT metric_kind, counter, COALESCE(last_reason,''), COALESCE(last_at,'') FROM worker_metric_state`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]WorkerMetricRow, 0)
	for rows.Next() {
		var kind, reason, at string
		var counter int64
		if err := rows.Scan(&kind, &counter, &reason, &at); err != nil {
			return nil, err
		}
		row := WorkerMetricRow{
			Kind:       WorkerMetricKind(kind),
			Counter:    uint64(counter),
			LastReason: reason,
		}
		if at != "" {
			if parsed, perr := time.Parse(time.RFC3339Nano, at); perr == nil {
				row.LastAt = parsed.UTC()
			}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *SQLiteWorkerMetricsRepository) IncrementSkip(
	ctx context.Context,
	kind WorkerMetricKind,
	reason string,
	at time.Time,
) error {
	atStr := at.UTC().Format(time.RFC3339Nano)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO worker_metric_state (metric_kind, counter, last_reason, last_at, updated_at)
		VALUES (?, 1, ?, ?, ?)
		ON CONFLICT(metric_kind) DO UPDATE SET
			counter = worker_metric_state.counter + 1,
			last_reason = excluded.last_reason,
			last_at = excluded.last_at,
			updated_at = excluded.updated_at
	`, string(kind), reason, atStr, atStr)
	return err
}

// PostgresWorkerMetricsRepository 提供 Postgres 后端的持久化实现。
type PostgresWorkerMetricsRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresWorkerMetricsRepository(pool *pgxpool.Pool) *PostgresWorkerMetricsRepository {
	return &PostgresWorkerMetricsRepository{pool: pool}
}

func (r *PostgresWorkerMetricsRepository) LoadAll(ctx context.Context) ([]WorkerMetricRow, error) {
	rows, err := r.pool.Query(ctx, `SELECT metric_kind, counter, COALESCE(last_reason,''), COALESCE(last_at, '0001-01-01 00:00:00'::timestamptz) FROM worker_metric_state`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]WorkerMetricRow, 0)
	for rows.Next() {
		var kind, reason string
		var counter int64
		var at time.Time
		if err := rows.Scan(&kind, &counter, &reason, &at); err != nil {
			return nil, err
		}
		row := WorkerMetricRow{
			Kind:       WorkerMetricKind(kind),
			Counter:    uint64(counter),
			LastReason: reason,
		}
		if !at.IsZero() && at.Year() > 1 {
			row.LastAt = at.UTC()
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return out, nil
}

func (r *PostgresWorkerMetricsRepository) IncrementSkip(
	ctx context.Context,
	kind WorkerMetricKind,
	reason string,
	at time.Time,
) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO worker_metric_state (metric_kind, counter, last_reason, last_at, updated_at)
		VALUES ($1, 1, $2, $3, $3)
		ON CONFLICT (metric_kind) DO UPDATE SET
			counter = worker_metric_state.counter + 1,
			last_reason = EXCLUDED.last_reason,
			last_at = EXCLUDED.last_at,
			updated_at = EXCLUDED.updated_at
	`, string(kind), reason, at.UTC())
	return err
}
