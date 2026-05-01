import { useMemo, useState } from 'react'
import { Coins, Minus, Plus, RefreshCcw, ShieldCheck } from 'lucide-react'
import {
  useCreditWallet,
  useDebitWallet,
  useWallet,
  useWalletTransactions,
} from '../../api/hooks'
import { useAuthStore } from '../../state/authStore'
import type { WalletKind, WalletTransaction } from '../../api/types'

const ADMIN_ROLES = new Set(['owner', 'admin'])

const KIND_LABEL: Record<WalletKind, string> = {
  credit: '充值',
  debit: '扣费',
  refund: '退款',
  adjust: '调整',
}

function formatNumber(value: number): string {
  return value.toLocaleString()
}

function formatDateTime(iso: string): string {
  const ts = Date.parse(iso)
  if (Number.isNaN(ts)) return iso
  return new Date(ts).toLocaleString()
}

function TransactionRow({ tx }: { tx: WalletTransaction }) {
  const positive = tx.direction > 0
  const sign = positive ? '+' : '−'
  return (
    <tr>
      <td>{formatDateTime(tx.created_at)}</td>
      <td>
        <span className={`wallet-kind-chip kind-${tx.kind}`}>{KIND_LABEL[tx.kind] ?? tx.kind}</span>
      </td>
      <td className={positive ? 'wallet-amount positive' : 'wallet-amount negative'}>
        {sign}
        {formatNumber(tx.amount)}
      </td>
      <td>{formatNumber(tx.balance_after)}</td>
      <td className="wallet-reason">{tx.reason ?? '—'}</td>
    </tr>
  )
}

export function WalletPage() {
  const session = useAuthStore((state) => state.session)
  const isAdmin = Boolean(session && ADMIN_ROLES.has(session.role))

  const walletQuery = useWallet(true)
  const txQuery = useWalletTransactions({ limit: 50, offset: 0 })
  const credit = useCreditWallet()
  const debit = useDebitWallet()

  const [amountInput, setAmountInput] = useState('')
  const [reasonInput, setReasonInput] = useState('')
  const [actionError, setActionError] = useState<string | null>(null)
  const [lastAction, setLastAction] = useState<string | null>(null)

  const balance = walletQuery.data?.wallet.balance ?? 0
  const updatedAt = walletQuery.data?.wallet.updated_at

  const transactions = useMemo(
    () => txQuery.data?.transactions ?? walletQuery.data?.recent_transactions ?? [],
    [txQuery.data, walletQuery.data],
  )

  const submit = async (kind: 'credit' | 'debit') => {
    setActionError(null)
    setLastAction(null)
    const amount = Number.parseInt(amountInput, 10)
    if (!Number.isFinite(amount) || amount <= 0) {
      setActionError('请输入大于 0 的整数金额')
      return
    }
    const request = { amount, reason: reasonInput.trim() || undefined }
    try {
      const mutator = kind === 'credit' ? credit : debit
      const tx = await mutator.mutateAsync(request)
      setAmountInput('')
      setReasonInput('')
      setLastAction(`已${KIND_LABEL[tx.kind] ?? kind} ${formatNumber(tx.amount)} 积分，新余额 ${formatNumber(tx.balance_after)}`)
      txQuery.refetch()
    } catch (err) {
      const message = (err as Error)?.message ?? '操作失败'
      if (message.includes('insufficient_balance') || message.includes('422')) {
        setActionError('余额不足，无法扣费')
      } else {
        setActionError(message)
      }
    }
  }

  return (
    <div className="admin-settings-page wallet-page">
      <header className="page-header">
        <Coins size={20} aria-hidden="true" />
        <h1>积分钱包</h1>
        <p className="page-subtitle">
          以组织为粒度的 AI 生产积分余额与流水。所有成员可查看，仅 owner / admin 可手动充值或扣费。
        </p>
      </header>

      <section className="provider-card" aria-label="余额">
        <div className="provider-card-header">
          <RefreshCcw size={18} aria-hidden="true" />
          <h2>当前余额</h2>
          <button
            className="action-btn secondary"
            type="button"
            disabled={walletQuery.isFetching}
            onClick={() => {
              walletQuery.refetch()
              txQuery.refetch()
            }}
          >
            {walletQuery.isFetching ? '刷新中…' : '立即刷新'}
          </button>
        </div>
        <div className="wallet-balance-grid">
          <article className="worker-metric-card has-skips">
            <span className="worker-metric-label">可用积分</span>
            <strong className="worker-metric-value">{formatNumber(balance)}</strong>
            <small className="worker-metric-hint">
              组织：{walletQuery.data?.wallet.organization_id ?? session?.organization_id ?? '—'}
            </small>
          </article>
          <article className="worker-metric-card">
            <span className="worker-metric-label">上次更新</span>
            <strong className="worker-metric-value">{updatedAt ? formatDateTime(updatedAt) : '—'}</strong>
            <small className="worker-metric-hint">
              {walletQuery.data ? '由系统在每次入账/出账时刷新' : '尚无任何流水'}
            </small>
          </article>
        </div>
        {walletQuery.isError ? (
          <p className="provider-card-body field-error">
            读取失败：{(walletQuery.error as Error)?.message ?? '未知错误'}
          </p>
        ) : null}
      </section>

      {isAdmin ? (
        <section className="provider-card" aria-label="手动充值/扣费">
          <div className="provider-card-header">
            <ShieldCheck size={18} aria-hidden="true" />
            <h2>手动调整（owner / admin）</h2>
          </div>
          <div className="wallet-action-form">
            <label>
              <span>金额（正整数）</span>
              <input
                type="number"
                min={1}
                step={1}
                value={amountInput}
                onChange={(event) => setAmountInput(event.target.value)}
                placeholder="例如 1000"
              />
            </label>
            <label>
              <span>备注（可选）</span>
              <input
                type="text"
                value={reasonInput}
                onChange={(event) => setReasonInput(event.target.value)}
                placeholder="例如：手动充值 / 试运行扣费"
                maxLength={200}
              />
            </label>
            <div className="wallet-action-buttons">
              <button
                className="action-btn primary"
                type="button"
                disabled={credit.isPending || debit.isPending}
                onClick={() => submit('credit')}
              >
                <Plus size={16} aria-hidden="true" /> 充值
              </button>
              <button
                className="action-btn secondary"
                type="button"
                disabled={credit.isPending || debit.isPending}
                onClick={() => submit('debit')}
              >
                <Minus size={16} aria-hidden="true" /> 扣费
              </button>
            </div>
            {actionError ? <p className="field-error">{actionError}</p> : null}
            {lastAction ? <p className="wallet-action-success">{lastAction}</p> : null}
          </div>
        </section>
      ) : (
        <section className="provider-card" aria-label="只读提示">
          <div className="provider-card-header">
            <ShieldCheck size={18} aria-hidden="true" />
            <h2>只读视图</h2>
          </div>
          <p className="provider-card-body">
            当前角色：<strong>{session?.role ?? '未知'}</strong>。如需调整余额，请联系 owner 或 admin。
          </p>
        </section>
      )}

      <section className="provider-card" aria-label="流水">
        <div className="provider-card-header">
          <Coins size={18} aria-hidden="true" />
          <h2>最近流水</h2>
          <button
            className="action-btn secondary"
            type="button"
            disabled={txQuery.isFetching}
            onClick={() => txQuery.refetch()}
          >
            {txQuery.isFetching ? '加载中…' : '刷新流水'}
          </button>
        </div>
        {transactions.length === 0 ? (
          <p className="provider-card-body">暂无流水。完成一次充值或扣费后会出现在这里。</p>
        ) : (
          <div className="wallet-table-wrapper">
            <table className="wallet-table">
              <thead>
                <tr>
                  <th>时间</th>
                  <th>类型</th>
                  <th>变动</th>
                  <th>余额</th>
                  <th>备注</th>
                </tr>
              </thead>
              <tbody>
                {transactions.map((tx) => (
                  <TransactionRow key={tx.id} tx={tx} />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  )
}
