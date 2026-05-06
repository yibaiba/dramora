import { AlertCircle, CheckCircle2, Coins, Gift, History, RefreshCcw, ShieldCheck } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Link } from 'react-router-dom'

import { useOperationCosts, useWallet } from '../../api/hooks'
import type { WalletTransaction } from '../../api/types'
import { StatePlaceholder } from '../components/StatePlaceholder'
import OperationCostsTable from '../components/OperationCostsTable'
import RedemptionCodeDialog from '../components/RedemptionCodeDialog'
import WalletBalanceCard from '../components/WalletBalanceCard'
import { studioRoutePaths } from '../routes'
import '../styles/wallet.css'

const EMPTY_TRANSACTIONS: WalletTransaction[] = []

export default function WalletPage() {
  const { data: wallet, isLoading: walletLoading, isFetching: walletFetching, refetch: refetchWallet } = useWallet()
  const { data: costs, isLoading: costsLoading } = useOperationCosts()
  const [isRedemptionDialogOpen, setIsRedemptionDialogOpen] = useState(false)
  const [paymentStatus, setPaymentStatus] = useState<'success' | 'cancel' | null>(() => getInitialPaymentStatus())
  const timerRef = useRef<number | null>(null)

  useEffect(() => {
    if (!paymentStatus) return

    if (paymentStatus === 'success') {
      refetchWallet()
    }

    timerRef.current = window.setTimeout(() => {
      window.history.replaceState({}, '', studioRoutePaths.wallet)
      setPaymentStatus(null)
    }, 5000)

    return () => {
      if (timerRef.current) {
        window.clearTimeout(timerRef.current)
      }
    }
  }, [paymentStatus, refetchWallet])

  const recentTransactions = wallet?.recent_transactions ?? EMPTY_TRANSACTIONS
  const summary = useMemo(() => {
    const balance = wallet?.wallet.balance ?? 0
    const highestCost = costs?.reduce((max, item) => (item.cost > max ? item.cost : max), 0) ?? 0
    return {
      balance,
      highestCost,
      recentCredits: recentTransactions.filter((item) => item.direction > 0).length,
      recentDebits: recentTransactions.filter((item) => item.direction < 0).length,
    }
  }, [costs, recentTransactions, wallet?.wallet.balance])

  if (walletLoading || costsLoading) {
    return <StatePlaceholder tone="loading" title="正在加载钱包数据..." description="读取余额、最近流水与操作定价。" />
  }

  return (
    <main className="wallet-page">
      <header className="page-header wallet-page-header">
        <div>
          <span className="section-kicker">计费中心</span>
          <h1>积分钱包</h1>
          <p className="page-subtitle">集中查看余额、最近扣费、充值入口与操作定价，确保生产流程不中断。</p>
        </div>
        <div className="wallet-page-actions">
          <button type="button" className="btn btn-secondary" onClick={() => refetchWallet()} disabled={walletFetching}>
            <RefreshCcw size={14} aria-hidden="true" />
            {walletFetching ? '刷新中...' : '刷新余额'}
          </button>
          <button type="button" className="btn btn-primary" onClick={() => setIsRedemptionDialogOpen(true)}>
            <Gift size={14} aria-hidden="true" />
            兑换赎回码
          </button>
        </div>
      </header>

      {paymentStatus ? (
        <section className={`wallet-banner ${paymentStatus === 'success' ? 'is-success' : 'is-warning'}`} aria-live="polite">
          {paymentStatus === 'success' ? <CheckCircle2 size={18} aria-hidden="true" /> : <AlertCircle size={18} aria-hidden="true" />}
          <div>
            <strong>{paymentStatus === 'success' ? '充值成功，余额已更新' : '充值流程已取消'}</strong>
            <span>
              {paymentStatus === 'success'
                ? 'Stripe 回调已返回，新的积分余额会自动刷新。'
                : '本次不会产生扣费，你可以稍后重新发起充值。'}
            </span>
          </div>
          <button type="button" className="btn btn-secondary" onClick={() => setPaymentStatus(null)}>
            关闭
          </button>
        </section>
      ) : null}

      <section className="wallet-hero">
        <WalletBalanceCard snapshot={wallet} />

        <article className="wallet-panel wallet-redeem-panel">
          <div className="wallet-panel-header">
            <div>
              <span className="section-kicker">兑换入口</span>
              <h2>活动码 / 赎回码</h2>
              <p>用于发放运营赠送积分、活动补贴或人工补偿，不影响正常 Stripe 充值流程。</p>
            </div>
            <Gift size={18} aria-hidden="true" />
          </div>
          <ul className="wallet-checklist">
            <li>支持大小写混输，提交前会自动标准化。</li>
            <li>兑换成功后立即刷新钱包余额与最近流水。</li>
            <li>无效、过期、已使用的 code 会给出明确错误提示。</li>
          </ul>
          <button type="button" className="btn btn-primary" onClick={() => setIsRedemptionDialogOpen(true)}>
            打开兑换窗口
          </button>
        </article>
      </section>

      <section className="wallet-summary-grid" aria-label="钱包概览">
        <article className="wallet-summary-card">
          <span className="wallet-summary-label">当前余额</span>
          <strong>{formatCredits(summary.balance)}</strong>
          <p>{summary.balance > 0 ? '当前仍可继续发起生产流程。' : '余额为 0，新的生成任务会被阻止。'}</p>
        </article>
        <article className="wallet-summary-card">
          <span className="wallet-summary-label">最近入账</span>
          <strong>{summary.recentCredits}</strong>
          <p>最近 {recentTransactions.length} 条流水中的入账事件数量。</p>
        </article>
        <article className="wallet-summary-card">
          <span className="wallet-summary-label">最近扣费</span>
          <strong>{summary.recentDebits}</strong>
          <p>帮助快速判断近期使用强度与消耗趋势。</p>
        </article>
        <article className="wallet-summary-card">
          <span className="wallet-summary-label">最高单次定价</span>
          <strong>{formatCredits(summary.highestCost)}</strong>
          <p>当前操作定价里最贵的一项，建议按此预估最小安全余额。</p>
        </article>
      </section>

      <section className="wallet-content-grid">
        <article className="wallet-panel">
          <div className="wallet-panel-header">
            <div>
              <span className="section-kicker">最近动态</span>
              <h2>最近流水</h2>
              <p>优先展示最新 5 条账单变化，便于确认充值和扣费是否符合预期。</p>
            </div>
            <Link className="btn btn-secondary" to={studioRoutePaths.transactions}>
              <History size={14} aria-hidden="true" />
              查看全部流水
            </Link>
          </div>

          {recentTransactions.length === 0 ? (
            <StatePlaceholder
              tone="empty"
              title="还没有钱包流水"
              description="充值、兑换或消费成功后，这里会显示最新的账单变化。"
              icon={Coins}
            />
          ) : (
            <ul className="wallet-transaction-list">
              {recentTransactions.slice(0, 5).map((transaction) => (
                <li key={transaction.id} className="wallet-transaction-row">
                  <div>
                    <div className="wallet-transaction-topline">
                      <strong>{getTransactionTitle(transaction)}</strong>
                      <span className={`wallet-transaction-amount ${transaction.direction > 0 ? 'is-positive' : 'is-negative'}`}>
                        {transaction.direction > 0 ? '+' : '-'}
                        {formatCredits(Math.abs(transaction.amount))}
                      </span>
                    </div>
                    <p>{getTransactionDescription(transaction)}</p>
                    <div className="wallet-transaction-meta">
                      <span>余额 {formatCredits(transaction.balance_after)}</span>
                      <span>{formatDateTime(transaction.created_at)}</span>
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </article>

        <article className="wallet-panel">
          <div className="wallet-panel-header">
            <div>
              <span className="section-kicker">计费地图</span>
              <h2>操作定价</h2>
              <p>所有能力按成功执行后扣费；失败请求不会消耗积分。</p>
            </div>
            <div className="wallet-pricing-note">
              <ShieldCheck size={14} aria-hidden="true" />
              自动扣费
            </div>
          </div>
          <OperationCostsTable costs={costs} />
        </article>
      </section>

      <RedemptionCodeDialog isOpen={isRedemptionDialogOpen} onClose={() => setIsRedemptionDialogOpen(false)} />
    </main>
  )
}

function getInitialPaymentStatus(): 'success' | 'cancel' | null {
  if (typeof window === 'undefined') return null
  const params = new URLSearchParams(window.location.search)
  const status = params.get('status')
  return status === 'success' || status === 'cancel' ? status : null
}

function formatCredits(value: number): string {
  return `${value.toLocaleString('zh-CN')} 积分`
}

function formatDateTime(value?: string): string {
  if (!value) return '—'
  return new Date(value).toLocaleString('zh-CN', {
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}

function getTransactionTitle(transaction: WalletTransaction): string {
  if (transaction.ref_type === 'redemption_code') return '赎回码兑换'
  switch (transaction.kind) {
    case 'credit':
      return '积分入账'
    case 'debit':
      return '积分扣费'
    case 'refund':
      return '积分退款'
    case 'adjust':
      return '余额调整'
    default:
      return transaction.kind
  }
}

function getTransactionDescription(transaction: WalletTransaction): string {
  if (transaction.reason) return transaction.reason
  if (transaction.ref_type === 'redemption_code') return '运营活动或人工发放的补贴积分。'
  return transaction.direction > 0 ? '成功充值或返还后增加到钱包。' : '一次成功的生产或运营动作导致扣费。'
}
