package repo

import "context"

type Tx interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type Transactor interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
