package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/yibaiba/dramora/internal/domain"
)

func (r *PostgresProductionRepository) CreateExport(
	ctx context.Context,
	params CreateExportParams,
) (domain.Export, error) {
	export, err := scanExport(r.pool.QueryRow(ctx, createExportSQL,
		params.ID, params.TimelineID, params.Status, params.Format,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Export{}, domain.ErrNotFound
	}
	return export, mapForeignKeyViolation(err)
}

func (r *PostgresProductionRepository) GetExport(ctx context.Context, exportID string) (domain.Export, error) {
	export, err := scanExport(r.pool.QueryRow(ctx, getExportSQL, exportID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Export{}, domain.ErrNotFound
	}
	return export, err
}

func (r *PostgresProductionRepository) ListExportsByStatus(
	ctx context.Context,
	status domain.ExportStatus,
	limit int,
) ([]domain.Export, error) {
	rows, err := r.pool.Query(ctx, listExportsByStatusSQL, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExports(rows)
}

func (r *PostgresProductionRepository) AdvanceExportStatus(
	ctx context.Context,
	params AdvanceExportStatusParams,
) (domain.Export, error) {
	export, err := scanExport(r.pool.QueryRow(ctx, advanceExportStatusSQL, params.ID, params.From, params.To))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Export{}, domain.ErrNotFound
	}
	return export, err
}
