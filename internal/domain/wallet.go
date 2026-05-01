package domain

import "time"

// WalletTransactionKind 枚举钱包流水类型。
type WalletTransactionKind string

const (
	// WalletKindCredit 表示余额增加（充值 / 赠送）。
	WalletKindCredit WalletTransactionKind = "credit"
	// WalletKindDebit 表示余额减少（消费）。
	WalletKindDebit WalletTransactionKind = "debit"
	// WalletKindRefund 表示因失败任务退还的余额（语义上等同 credit，单独一类便于审计）。
	WalletKindRefund WalletTransactionKind = "refund"
	// WalletKindAdjust 表示运营手动调账（可正可负，sign 由 Amount 控制层决定）。
	WalletKindAdjust WalletTransactionKind = "adjust"
)

// Wallet 描述一个组织当前的余额状态。
type Wallet struct {
	OrganizationID string
	Balance        int64
	UpdatedAt      time.Time
}

// WalletTransaction 描述一次余额变动。
// Amount 始终为正整数；增减方向由 Kind 决定（credit/refund 增；debit 减；adjust 由 Direction 决定）。
type WalletTransaction struct {
	ID             string
	OrganizationID string
	Kind           WalletTransactionKind
	// Direction 仅在 Kind=adjust 时使用：+1 增 / -1 减。其它 kind 由 Kind 自身派生。
	Direction    int
	Amount       int64
	Reason       string
	RefType      string
	RefID        string
	BalanceAfter int64
	ActorUserID  string
	CreatedAt    time.Time
}

// IsValidWalletKind 校验外部输入的 kind 字符串是否合法。
func IsValidWalletKind(s string) bool {
	switch WalletTransactionKind(s) {
	case WalletKindCredit, WalletKindDebit, WalletKindRefund, WalletKindAdjust:
		return true
	default:
		return false
	}
}
