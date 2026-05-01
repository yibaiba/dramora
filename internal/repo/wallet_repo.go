package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yibaiba/dramora/internal/domain"
)

// ErrInsufficientBalance 表示余额不足以完成 debit。
var ErrInsufficientBalance = errors.New("wallet: insufficient balance")

// WalletApplyParams 描述一次余额变动请求。
// Direction：+1 增 / -1 减。Amount 必须为正整数。
type WalletApplyParams struct {
	OrganizationID string
	Kind           domain.WalletTransactionKind
	Direction      int
	Amount         int64
	Reason         string
	RefType        string
	RefID          string
	ActorUserID    string
	TransactionID  string
	CreatedAt      time.Time
}

// WalletTransactionFilter 用于历史流水分页查询。
type WalletTransactionFilter struct {
	OrganizationID string
	Kinds          []string
	Limit          int
	Offset         int
}

// WalletTransactionPage 是流水分页结果。
type WalletTransactionPage struct {
	Transactions []domain.WalletTransaction
	HasMore      bool
}

// WalletRepository 抽象钱包余额与流水的持久化层。
// ApplyTransaction 必须原子完成 "读余额 / 校验 / 写余额 / 落流水"。
type WalletRepository interface {
	GetWallet(ctx context.Context, organizationID string) (domain.Wallet, error)
	ApplyTransaction(ctx context.Context, params WalletApplyParams) (domain.Wallet, domain.WalletTransaction, error)
	ListTransactions(ctx context.Context, filter WalletTransactionFilter) (WalletTransactionPage, error)
}

// MemoryWalletRepository 提供进程内实现。
type MemoryWalletRepository struct {
	mu      sync.Mutex
	wallets map[string]domain.Wallet
	txs     []domain.WalletTransaction
}

func NewMemoryWalletRepository() *MemoryWalletRepository {
	return &MemoryWalletRepository{wallets: make(map[string]domain.Wallet)}
}

func (r *MemoryWalletRepository) GetWallet(_ context.Context, orgID string) (domain.Wallet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if w, ok := r.wallets[orgID]; ok {
		return w, nil
	}
	return domain.Wallet{OrganizationID: orgID, Balance: 0, UpdatedAt: time.Time{}}, nil
}

func (r *MemoryWalletRepository) ApplyTransaction(
	_ context.Context,
	params WalletApplyParams,
) (domain.Wallet, domain.WalletTransaction, error) {
	if params.Amount <= 0 {
		return domain.Wallet{}, domain.WalletTransaction{}, fmt.Errorf("wallet: amount must be positive")
	}
	if params.Direction != 1 && params.Direction != -1 {
		return domain.Wallet{}, domain.WalletTransaction{}, fmt.Errorf("wallet: direction must be +1 or -1")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current := r.wallets[params.OrganizationID]
	delta := params.Amount * int64(params.Direction)
	newBalance := current.Balance + delta
	if newBalance < 0 {
		return domain.Wallet{}, domain.WalletTransaction{}, ErrInsufficientBalance
	}
	createdAt := params.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updated := domain.Wallet{
		OrganizationID: params.OrganizationID,
		Balance:        newBalance,
		UpdatedAt:      createdAt,
	}
	r.wallets[params.OrganizationID] = updated
	tx := domain.WalletTransaction{
		ID:             params.TransactionID,
		OrganizationID: params.OrganizationID,
		Kind:           params.Kind,
		Direction:      params.Direction,
		Amount:         params.Amount,
		Reason:         params.Reason,
		RefType:        params.RefType,
		RefID:          params.RefID,
		BalanceAfter:   newBalance,
		ActorUserID:    params.ActorUserID,
		CreatedAt:      createdAt,
	}
	r.txs = append(r.txs, tx)
	return updated, tx, nil
}

func (r *MemoryWalletRepository) ListTransactions(
	_ context.Context,
	filter WalletTransactionFilter,
) (WalletTransactionPage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	kindSet := stringSet(filter.Kinds)
	matched := make([]domain.WalletTransaction, 0)
	for _, tx := range r.txs {
		if filter.OrganizationID != "" && tx.OrganizationID != filter.OrganizationID {
			continue
		}
		if len(kindSet) > 0 && !kindSet[string(tx.Kind)] {
			continue
		}
		matched = append(matched, tx)
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i].CreatedAt.After(matched[j].CreatedAt) })
	limit := filter.Limit
	if limit <= 0 {
		limit = len(matched)
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(matched) {
		return WalletTransactionPage{Transactions: []domain.WalletTransaction{}}, nil
	}
	end := offset + limit
	hasMore := false
	if end < len(matched) {
		hasMore = true
	} else {
		end = len(matched)
	}
	return WalletTransactionPage{
		Transactions: append([]domain.WalletTransaction(nil), matched[offset:end]...),
		HasMore:      hasMore,
	}, nil
}

// SQLiteWalletRepository 复用 sqlite_migrations 中的 wallets / wallet_transactions 表。
type SQLiteWalletRepository struct {
	db *sql.DB
}

func NewSQLiteWalletRepository(db *sql.DB) *SQLiteWalletRepository {
	return &SQLiteWalletRepository{db: db}
}

func (r *SQLiteWalletRepository) GetWallet(ctx context.Context, orgID string) (domain.Wallet, error) {
	row := r.db.QueryRowContext(ctx, `SELECT organization_id, balance, updated_at FROM wallets WHERE organization_id = ?`, orgID)
	var (
		id        string
		balance   int64
		updatedAt sql.NullString
	)
	if err := row.Scan(&id, &balance, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Wallet{OrganizationID: orgID, Balance: 0}, nil
		}
		return domain.Wallet{}, fmt.Errorf("get wallet: %w", err)
	}
	w := domain.Wallet{OrganizationID: id, Balance: balance}
	if updatedAt.Valid {
		if t, err := time.Parse("2006-01-02T15:04:05.000Z", updatedAt.String); err == nil {
			w.UpdatedAt = t
		}
	}
	return w, nil
}

func (r *SQLiteWalletRepository) ApplyTransaction(
	ctx context.Context,
	params WalletApplyParams,
) (domain.Wallet, domain.WalletTransaction, error) {
	if params.Amount <= 0 {
		return domain.Wallet{}, domain.WalletTransaction{}, fmt.Errorf("wallet: amount must be positive")
	}
	if params.Direction != 1 && params.Direction != -1 {
		return domain.Wallet{}, domain.WalletTransaction{}, fmt.Errorf("wallet: direction must be +1 or -1")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Wallet{}, domain.WalletTransaction{}, fmt.Errorf("wallet begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var current int64
	row := tx.QueryRowContext(ctx, `SELECT balance FROM wallets WHERE organization_id = ?`, params.OrganizationID)
	if err := row.Scan(&current); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return domain.Wallet{}, domain.WalletTransaction{}, fmt.Errorf("wallet read: %w", err)
		}
		current = 0
	}
	delta := params.Amount * int64(params.Direction)
	newBalance := current + delta
	if newBalance < 0 {
		return domain.Wallet{}, domain.WalletTransaction{}, ErrInsufficientBalance
	}
	createdAt := params.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	createdStr := createdAt.UTC().Format("2006-01-02T15:04:05.000Z")
	if _, err := tx.ExecContext(ctx, `INSERT INTO wallets (organization_id, balance, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(organization_id) DO UPDATE SET balance = excluded.balance, updated_at = excluded.updated_at`,
		params.OrganizationID, newBalance, createdStr); err != nil {
		return domain.Wallet{}, domain.WalletTransaction{}, fmt.Errorf("wallet upsert: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO wallet_transactions
		(id, organization_id, kind, direction, amount, reason, ref_type, ref_id, balance_after, actor_user_id, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		params.TransactionID, params.OrganizationID, string(params.Kind), params.Direction, params.Amount,
		nullableString(params.Reason), nullableString(params.RefType), nullableString(params.RefID),
		newBalance, nullableString(params.ActorUserID), createdStr); err != nil {
		return domain.Wallet{}, domain.WalletTransaction{}, fmt.Errorf("wallet append tx: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Wallet{}, domain.WalletTransaction{}, fmt.Errorf("wallet commit: %w", err)
	}
	w := domain.Wallet{OrganizationID: params.OrganizationID, Balance: newBalance, UpdatedAt: createdAt}
	dt := domain.WalletTransaction{
		ID:             params.TransactionID,
		OrganizationID: params.OrganizationID,
		Kind:           params.Kind,
		Direction:      params.Direction,
		Amount:         params.Amount,
		Reason:         params.Reason,
		RefType:        params.RefType,
		RefID:          params.RefID,
		BalanceAfter:   newBalance,
		ActorUserID:    params.ActorUserID,
		CreatedAt:      createdAt,
	}
	return w, dt, nil
}

func (r *SQLiteWalletRepository) ListTransactions(
	ctx context.Context,
	filter WalletTransactionFilter,
) (WalletTransactionPage, error) {
	var (
		conds []string
		args  []any
	)
	if filter.OrganizationID != "" {
		conds = append(conds, "organization_id = ?")
		args = append(args, filter.OrganizationID)
	}
	if len(filter.Kinds) > 0 {
		ph := make([]string, 0, len(filter.Kinds))
		for _, k := range filter.Kinds {
			ph = append(ph, "?")
			args = append(args, k)
		}
		conds = append(conds, "kind IN ("+strings.Join(ph, ",")+")")
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit+1, offset)
	q := fmt.Sprintf(`SELECT id, organization_id, kind, direction, amount, reason, ref_type, ref_id, balance_after, actor_user_id, created_at
		FROM wallet_transactions %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, where)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return WalletTransactionPage{}, fmt.Errorf("list wallet tx: %w", err)
	}
	defer rows.Close()
	var out []domain.WalletTransaction
	for rows.Next() {
		var (
			tx        domain.WalletTransaction
			reason    sql.NullString
			refType   sql.NullString
			refID     sql.NullString
			actor     sql.NullString
			createdAt string
			kind      string
		)
		if err := rows.Scan(&tx.ID, &tx.OrganizationID, &kind, &tx.Direction, &tx.Amount,
			&reason, &refType, &refID, &tx.BalanceAfter, &actor, &createdAt); err != nil {
			return WalletTransactionPage{}, fmt.Errorf("scan wallet tx: %w", err)
		}
		tx.Kind = domain.WalletTransactionKind(kind)
		tx.Reason = reason.String
		tx.RefType = refType.String
		tx.RefID = refID.String
		tx.ActorUserID = actor.String
		if t, err := time.Parse("2006-01-02T15:04:05.000Z", createdAt); err == nil {
			tx.CreatedAt = t
		} else if t, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
			tx.CreatedAt = t
		}
		out = append(out, tx)
	}
	if err := rows.Err(); err != nil {
		return WalletTransactionPage{}, fmt.Errorf("iterate wallet tx: %w", err)
	}
	hasMore := false
	if len(out) > limit {
		hasMore = true
		out = out[:limit]
	}
	return WalletTransactionPage{Transactions: out, HasMore: hasMore}, nil
}

// PostgresWalletRepository 提供 Postgres 后端实现，使用 SELECT ... FOR UPDATE 串行化余额变动。
type PostgresWalletRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresWalletRepository(pool *pgxpool.Pool) *PostgresWalletRepository {
	return &PostgresWalletRepository{pool: pool}
}

func (r *PostgresWalletRepository) GetWallet(ctx context.Context, orgID string) (domain.Wallet, error) {
	var (
		id        string
		balance   int64
		updatedAt time.Time
	)
	err := r.pool.QueryRow(ctx, `SELECT organization_id, balance, updated_at FROM wallets WHERE organization_id = $1`, orgID).
		Scan(&id, &balance, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "no rows") {
			return domain.Wallet{OrganizationID: orgID, Balance: 0}, nil
		}
		return domain.Wallet{}, fmt.Errorf("get wallet: %w", err)
	}
	return domain.Wallet{OrganizationID: id, Balance: balance, UpdatedAt: updatedAt.UTC()}, nil
}

func (r *PostgresWalletRepository) ApplyTransaction(
	ctx context.Context,
	params WalletApplyParams,
) (domain.Wallet, domain.WalletTransaction, error) {
	if params.Amount <= 0 {
		return domain.Wallet{}, domain.WalletTransaction{}, fmt.Errorf("wallet: amount must be positive")
	}
	if params.Direction != 1 && params.Direction != -1 {
		return domain.Wallet{}, domain.WalletTransaction{}, fmt.Errorf("wallet: direction must be +1 or -1")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Wallet{}, domain.WalletTransaction{}, fmt.Errorf("wallet pg begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var current int64
	row := tx.QueryRow(ctx, `SELECT balance FROM wallets WHERE organization_id = $1 FOR UPDATE`, params.OrganizationID)
	if err := row.Scan(&current); err != nil {
		if !errors.Is(err, sql.ErrNoRows) && !strings.Contains(err.Error(), "no rows") {
			return domain.Wallet{}, domain.WalletTransaction{}, fmt.Errorf("wallet pg read: %w", err)
		}
		current = 0
	}
	delta := params.Amount * int64(params.Direction)
	newBalance := current + delta
	if newBalance < 0 {
		return domain.Wallet{}, domain.WalletTransaction{}, ErrInsufficientBalance
	}
	createdAt := params.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	if _, err := tx.Exec(ctx, `INSERT INTO wallets (organization_id, balance, updated_at)
		VALUES ($1,$2,$3)
		ON CONFLICT (organization_id) DO UPDATE SET balance = EXCLUDED.balance, updated_at = EXCLUDED.updated_at`,
		params.OrganizationID, newBalance, createdAt); err != nil {
		return domain.Wallet{}, domain.WalletTransaction{}, fmt.Errorf("wallet pg upsert: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO wallet_transactions
		(id, organization_id, kind, direction, amount, reason, ref_type, ref_id, balance_after, actor_user_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		params.TransactionID, params.OrganizationID, string(params.Kind), params.Direction, params.Amount,
		nullableString(params.Reason), nullableString(params.RefType), nullableString(params.RefID),
		newBalance, nullableString(params.ActorUserID), createdAt); err != nil {
		return domain.Wallet{}, domain.WalletTransaction{}, fmt.Errorf("wallet pg append tx: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Wallet{}, domain.WalletTransaction{}, fmt.Errorf("wallet pg commit: %w", err)
	}
	w := domain.Wallet{OrganizationID: params.OrganizationID, Balance: newBalance, UpdatedAt: createdAt}
	dt := domain.WalletTransaction{
		ID:             params.TransactionID,
		OrganizationID: params.OrganizationID,
		Kind:           params.Kind,
		Direction:      params.Direction,
		Amount:         params.Amount,
		Reason:         params.Reason,
		RefType:        params.RefType,
		RefID:          params.RefID,
		BalanceAfter:   newBalance,
		ActorUserID:    params.ActorUserID,
		CreatedAt:      createdAt,
	}
	return w, dt, nil
}

func (r *PostgresWalletRepository) ListTransactions(
	ctx context.Context,
	filter WalletTransactionFilter,
) (WalletTransactionPage, error) {
	var (
		conds []string
		args  []any
		i     = 1
	)
	if filter.OrganizationID != "" {
		conds = append(conds, fmt.Sprintf("organization_id = $%d", i))
		args = append(args, filter.OrganizationID)
		i++
	}
	if len(filter.Kinds) > 0 {
		ph := make([]string, 0, len(filter.Kinds))
		for _, k := range filter.Kinds {
			ph = append(ph, fmt.Sprintf("$%d", i))
			args = append(args, k)
			i++
		}
		conds = append(conds, "kind IN ("+strings.Join(ph, ",")+")")
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit+1, offset)
	q := fmt.Sprintf(`SELECT id, organization_id, kind, direction, amount,
		COALESCE(reason,''), COALESCE(ref_type,''), COALESCE(ref_id,''),
		balance_after, COALESCE(actor_user_id,''), created_at
		FROM wallet_transactions %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, i, i+1)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return WalletTransactionPage{}, fmt.Errorf("list wallet tx pg: %w", err)
	}
	defer rows.Close()
	var out []domain.WalletTransaction
	for rows.Next() {
		var (
			tx   domain.WalletTransaction
			kind string
			at   time.Time
		)
		if err := rows.Scan(&tx.ID, &tx.OrganizationID, &kind, &tx.Direction, &tx.Amount,
			&tx.Reason, &tx.RefType, &tx.RefID, &tx.BalanceAfter, &tx.ActorUserID, &at); err != nil {
			return WalletTransactionPage{}, fmt.Errorf("scan wallet tx pg: %w", err)
		}
		tx.Kind = domain.WalletTransactionKind(kind)
		tx.CreatedAt = at.UTC()
		out = append(out, tx)
	}
	if err := rows.Err(); err != nil {
		return WalletTransactionPage{}, fmt.Errorf("iterate wallet tx pg: %w", err)
	}
	hasMore := false
	if len(out) > limit {
		hasMore = true
		out = out[:limit]
	}
	return WalletTransactionPage{Transactions: out, HasMore: hasMore}, nil
}
