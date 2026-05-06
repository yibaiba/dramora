import { AlertCircle, CheckCircle2, Gift, Loader2, X } from 'lucide-react'
import { useState } from 'react'

import type { RedeemCodeResponse } from '../../api/types'
import { useRedeemCode } from '../../api/hooks'
import { normalizeRedemptionCode, validateRedemptionCode } from '../../lib/redemption'

interface RedemptionCodeDialogProps {
  isOpen: boolean
  onClose: () => void
}

interface ApiError {
  response?: {
    data?: {
      error?: string
      message?: string
    }
  }
}

export default function RedemptionCodeDialog({ isOpen, onClose }: RedemptionCodeDialogProps) {
  const [code, setCode] = useState('')
  const [feedback, setFeedback] = useState<{
    details?: RedeemCodeResponse
    message: string
    type: 'success' | 'error'
  } | null>(null)

  const redeemCodeMutation = useRedeemCode()

  const closeDialog = () => {
    setCode('')
    setFeedback(null)
    onClose()
  }

  const handleRedeem = async () => {
    const normalized = normalizeRedemptionCode(code)
    const validation = validateRedemptionCode(normalized)

    if (!validation.valid) {
      setFeedback({ message: validation.error || '赎回码格式不正确', type: 'error' })
      return
    }

    try {
      const response = await redeemCodeMutation.mutateAsync({ code: normalized })
      setFeedback({
        details: response,
        message: response.message,
        type: 'success',
      })
      setCode('')
    } catch (error) {
      const apiError = error as ApiError
      const errorCode = apiError.response?.data?.error
      setFeedback({
        message:
          errorCode === 'code_not_found'
            ? '赎回码不存在'
            : errorCode === 'code_already_used'
              ? '赎回码已被使用'
              : errorCode === 'code_expired'
                ? '赎回码已过期'
                : apiError.response?.data?.message || '兑换失败，请检查输入后重试',
        type: 'error',
      })
    }
  }

  if (!isOpen) return null

  return (
    <div className="wallet-dialog-backdrop" role="presentation">
      <div className="wallet-dialog" role="dialog" aria-modal="true" aria-labelledby="wallet-redeem-title">
        <div className="wallet-dialog-header">
          <div>
            <span className="section-kicker">兑换入口</span>
            <h2 id="wallet-redeem-title">兑换赎回码</h2>
            <p>活动赠送、运营补贴或手动发放的积分会通过这里入账。</p>
          </div>
          <button type="button" className="btn btn-secondary wallet-dialog-close" onClick={closeDialog} aria-label="关闭兑换对话框">
            <X size={16} aria-hidden="true" />
          </button>
        </div>

        <div className="wallet-dialog-body">
          {feedback ? (
            <div className={`wallet-feedback-card ${feedback.type === 'success' ? 'is-success' : 'is-error'}`} role={feedback.type === 'error' ? 'alert' : 'status'}>
              {feedback.type === 'success' ? <CheckCircle2 size={18} aria-hidden="true" /> : <AlertCircle size={18} aria-hidden="true" />}
              <div>
                <strong>{feedback.message}</strong>
                {feedback.details ? (
                  <p>
                    本次到账 {feedback.details.amount.toLocaleString('zh-CN')} 积分，当前余额 {feedback.details.newBalance.toLocaleString('zh-CN')} 积分。
                  </p>
                ) : null}
              </div>
            </div>
          ) : null}

          <label className="wallet-form-field">
            <span>赎回码</span>
            <input
              type="text"
              value={code}
              onChange={(event) => setCode(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' && !redeemCodeMutation.isPending) {
                  void handleRedeem()
                }
              }}
              placeholder="例如 DRAMORA-2025-XXXX"
              disabled={redeemCodeMutation.isPending}
              autoFocus
            />
            <small>输入时会自动去除空格并标准化大小写。</small>
          </label>

          <div className="wallet-inline-note">
            <Gift size={16} aria-hidden="true" />
            <span>每个赎回码只能成功使用一次；成功后钱包余额与最近流水会自动刷新。</span>
          </div>
        </div>

        <div className="wallet-dialog-footer">
          <button type="button" className="btn btn-secondary" onClick={closeDialog} disabled={redeemCodeMutation.isPending}>
            {feedback?.type === 'success' ? '完成' : '取消'}
          </button>
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => void handleRedeem()}
            disabled={redeemCodeMutation.isPending || !code.trim()}
          >
            {redeemCodeMutation.isPending ? <Loader2 size={16} className="animate-spin" aria-hidden="true" /> : null}
            {redeemCodeMutation.isPending ? '兑换中...' : '确认兑换'}
          </button>
        </div>
      </div>
    </div>
  )
}
