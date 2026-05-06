import { CreditCard, Loader2, X } from 'lucide-react'
import { useState } from 'react'

import { useInitiateChargeWallet } from '../../api/hooks'

interface ChargeWalletDialogProps {
  isOpen: boolean
  onClose: () => void
}

const PRESET_AMOUNTS = [100, 500, 1000, 3000]

export default function ChargeWalletDialog({ isOpen, onClose }: ChargeWalletDialogProps) {
  const [amount, setAmount] = useState('100')
  const [error, setError] = useState<string | null>(null)
  const initiateChargeMutation = useInitiateChargeWallet()

  const closeDialog = () => {
    setError(null)
    onClose()
  }

  const handleCharge = async () => {
    const chargeAmount = Number.parseInt(amount, 10)
    if (!Number.isFinite(chargeAmount) || chargeAmount <= 0) {
      setError('请输入大于 0 的充值积分数量。')
      return
    }

    setError(null)

    try {
      const response = await initiateChargeMutation.mutateAsync({
        amount: chargeAmount,
        currency: 'USD',
      })

      if (response.url) {
        window.location.href = response.url
        return
      }

      setError('未拿到 Stripe 支付页地址，请稍后重试。')
    } catch (chargeError) {
      setError(chargeError instanceof Error ? chargeError.message : '发起充值失败，请稍后重试。')
    }
  }

  if (!isOpen) return null

  return (
    <div className="wallet-dialog-backdrop" role="presentation">
      <div className="wallet-dialog" role="dialog" aria-modal="true" aria-labelledby="wallet-charge-title">
        <div className="wallet-dialog-header">
          <div>
            <span className="section-kicker">充值入口</span>
            <h2 id="wallet-charge-title">充值积分</h2>
            <p>将跳转到 Stripe 支付页完成安全支付，成功后自动返回钱包页面。</p>
          </div>
          <button type="button" className="btn btn-secondary wallet-dialog-close" onClick={closeDialog} aria-label="关闭充值对话框">
            <X size={16} aria-hidden="true" />
          </button>
        </div>

        <div className="wallet-dialog-body">
          <div className="wallet-amount-presets" role="list" aria-label="快捷充值额度">
            {PRESET_AMOUNTS.map((preset) => (
              <button
                type="button"
                key={preset}
                className={amount === String(preset) ? 'is-active' : ''}
                onClick={() => setAmount(String(preset))}
              >
                {preset.toLocaleString('zh-CN')} 积分
              </button>
            ))}
          </div>

          <label className="wallet-form-field">
            <span>自定义金额</span>
            <input
              type="number"
              value={amount}
              onChange={(event) => setAmount(event.target.value)}
              min="1"
              step="1"
              disabled={initiateChargeMutation.isPending}
            />
            <small>至少充值 1 积分；建议覆盖一次完整的视频生产预算。</small>
          </label>

          {error ? (
            <div className="wallet-feedback-card is-error" role="alert">
              <X size={18} aria-hidden="true" />
              <div>
                <strong>无法发起充值</strong>
                <p>{error}</p>
              </div>
            </div>
          ) : null}

          <div className="wallet-inline-note">
            <CreditCard size={16} aria-hidden="true" />
            <span>支付流程由 Stripe 托管；返回后 URL 会带上 `status=success | cancel` 作为回调结果。</span>
          </div>
        </div>

        <div className="wallet-dialog-footer">
          <button type="button" className="btn btn-secondary" onClick={closeDialog} disabled={initiateChargeMutation.isPending}>
            取消
          </button>
          <button type="button" className="btn btn-primary" onClick={() => void handleCharge()} disabled={initiateChargeMutation.isPending}>
            {initiateChargeMutation.isPending ? <Loader2 size={16} className="animate-spin" aria-hidden="true" /> : null}
            {initiateChargeMutation.isPending ? '跳转中...' : '前往 Stripe 支付页'}
          </button>
        </div>
      </div>
    </div>
  )
}
