import { Coins, TrendingUp } from 'lucide-react'
import { useState } from 'react'

import type { WalletSnapshot } from '../../api/types'
import ChargeWalletDialog from './ChargeWalletDialog'

interface WalletBalanceCardProps {
  snapshot?: WalletSnapshot
}

export default function WalletBalanceCard({ snapshot }: WalletBalanceCardProps) {
  const [isChargeDialogOpen, setIsChargeDialogOpen] = useState(false)
  const balance = snapshot?.wallet.balance ?? 0
  const lastUpdated = snapshot?.wallet.updated_at ? formatDateTime(snapshot.wallet.updated_at) : '尚未更新'
  const lastTransaction = snapshot?.recent_transactions[0]

  return (
    <>
      <article className="wallet-balance-card">
        <div className="wallet-balance-header">
          <div>
            <span className="section-kicker">余额总览</span>
            <h2>可用余额</h2>
            <p>成功执行的故事分析、聊天、图像和视频生成会自动从此钱包扣费。</p>
          </div>
          <div className="wallet-balance-icon" aria-hidden="true">
            <Coins size={22} />
          </div>
        </div>

        <div className="wallet-balance-value">
          <strong>{balance.toLocaleString('zh-CN')}</strong>
          <span>积分</span>
        </div>

        <div className="wallet-balance-metrics">
          <div>
            <span>状态</span>
            <strong>{balance > 0 ? '可继续生产' : '需要充值'}</strong>
          </div>
          <div>
            <span>最近更新</span>
            <strong>{lastUpdated}</strong>
          </div>
        </div>

        <div className="wallet-balance-callout">
          <span className="wallet-balance-callout-label">最近一笔</span>
          <strong>{lastTransaction ? `${lastTransaction.direction > 0 ? '+' : '-'}${Math.abs(lastTransaction.amount)} 积分` : '暂无流水'}</strong>
          <p>{lastTransaction?.reason || '充值、兑换或消费成功后会在这里出现摘要。'}</p>
        </div>

        <button type="button" className="btn btn-primary wallet-balance-action" onClick={() => setIsChargeDialogOpen(true)}>
          <TrendingUp size={16} aria-hidden="true" />
          立即充值
        </button>
      </article>

      <ChargeWalletDialog isOpen={isChargeDialogOpen} onClose={() => setIsChargeDialogOpen(false)} />
    </>
  )
}

function formatDateTime(value: string): string {
  return new Date(value).toLocaleString('zh-CN', {
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}
