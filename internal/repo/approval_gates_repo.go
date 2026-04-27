package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/yibaiba/dramora/internal/domain"
)

func (r *PostgresProductionRepository) ListApprovalGates(
	ctx context.Context,
	episodeID string,
) ([]domain.ApprovalGate, error) {
	rows, err := r.pool.Query(ctx, listApprovalGatesSQL, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApprovalGates(rows)
}

func (r *PostgresProductionRepository) GetApprovalGate(
	ctx context.Context,
	gateID string,
) (domain.ApprovalGate, error) {
	gate, err := scanApprovalGate(r.pool.QueryRow(ctx, getApprovalGateSQL, gateID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ApprovalGate{}, domain.ErrNotFound
	}
	return gate, err
}

func (r *PostgresProductionRepository) SaveApprovalGate(
	ctx context.Context,
	params SaveApprovalGateParams,
) (domain.ApprovalGate, error) {
	gate, err := scanApprovalGate(r.pool.QueryRow(ctx, upsertApprovalGateSQL,
		params.ID, params.ProjectID, params.EpisodeID, nullableUUID(params.WorkflowRunID),
		params.GateType, params.SubjectType, params.SubjectID, params.Status,
	))
	return gate, mapForeignKeyViolation(err)
}

func (r *PostgresProductionRepository) ReviewApprovalGate(
	ctx context.Context,
	params ReviewApprovalGateParams,
) (domain.ApprovalGate, error) {
	gate, err := scanApprovalGate(r.pool.QueryRow(ctx, reviewApprovalGateSQL,
		params.ID, params.Status, params.ReviewedBy, params.ReviewNote,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ApprovalGate{}, domain.ErrNotFound
	}
	return gate, err
}
