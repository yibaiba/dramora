package cost

import "context"

type Reservation struct {
	ID             string
	BudgetID       string
	EstimatedCents int64
	CommittedCents int64
	Released       bool
}

type Service interface {
	Reserve(ctx context.Context, budgetID string, estimatedCents int64) (Reservation, error)
	Commit(ctx context.Context, reservationID string, actualCents int64) error
	Release(ctx context.Context, reservationID string) error
}
