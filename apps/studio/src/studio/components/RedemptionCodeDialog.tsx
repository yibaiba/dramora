import { useState } from 'react'
import { AlertCircle, CheckCircle, Loader2, X } from 'lucide-react'
import { useRedeemCode } from '../../api/hooks'
import type { RedeemCodeResponse } from '../../api/types'

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
    type: 'success' | 'error'
    message: string
    details?: RedeemCodeResponse
  } | null>(null)

  const { mutate: redeem, isPending } = useRedeemCode()

  const handleRedeem = () => {
    const trimmedCode = code.trim().toUpperCase()
    if (!trimmedCode) {
      setFeedback({
        type: 'error',
        message: '请输入赎回码',
      })
      return
    }

    redeem(
      { code: trimmedCode },
      {
        onSuccess: (response) => {
          setFeedback({
            type: 'success',
            message: response.message,
            details: response,
          })
          setCode('')
          // Auto-close after 3 seconds
          setTimeout(() => {
            onClose()
            setFeedback(null)
          }, 3000)
        },
        onError: (error) => {
          const apiError = error as ApiError
          const errorMessage =
            apiError?.response?.data?.error === 'code_not_found'
              ? '赎回码不存在'
              : apiError?.response?.data?.error === 'code_already_used'
                ? '赎回码已被使用'
                : apiError?.response?.data?.error === 'code_expired'
                  ? '赎回码已过期'
                  : apiError?.response?.data?.message || '兑换失败，请检查输入'

          setFeedback({
            type: 'error',
            message: errorMessage,
          })
        },
      }
    )
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-slate-900 rounded-lg shadow-lg max-w-md w-full mx-4">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-slate-200 dark:border-slate-700">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-white">兑换赎回码</h2>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6">
          {feedback ? (
            <div
              className={`rounded-lg p-4 flex gap-3 ${
                feedback.type === 'success'
                  ? 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800'
                  : 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800'
              }`}
            >
              {feedback.type === 'success' ? (
                <CheckCircle className="w-5 h-5 text-green-600 dark:text-green-400 flex-shrink-0 mt-0.5" />
              ) : (
                <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
              )}
              <div className="flex-1">
                <p
                  className={`font-semibold mb-1 ${
                    feedback.type === 'success'
                      ? 'text-green-900 dark:text-green-200'
                      : 'text-red-900 dark:text-red-200'
                  }`}
                >
                  {feedback.message}
                </p>
                {feedback.details && (
                  <div
                    className={`text-sm space-y-1 ${
                      feedback.type === 'success'
                        ? 'text-green-700 dark:text-green-300'
                        : 'text-red-700 dark:text-red-300'
                    }`}
                  >
                    <p>获得积分：+{feedback.details.amount}</p>
                    <p>新余额：{feedback.details.newBalance}</p>
                  </div>
                )}
              </div>
            </div>
          ) : (
            <>
              <div className="mb-4">
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  赎回码
                </label>
                <input
                  type="text"
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !isPending) {
                      handleRedeem()
                    }
                  }}
                  placeholder="输入你的赎回码（大小写不敏感）"
                  disabled={isPending}
                  className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  autoFocus
                />
                <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                  输入框会自动转换为大写字母并去除空格
                </p>
              </div>

              <div className="p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg mb-4">
                <p className="text-sm text-blue-700 dark:text-blue-300">
                  输入你的赎回码来获得积分。每个赎回码只能使用一次。
                </p>
              </div>
            </>
          )}
        </div>

        {/* Footer */}
        <div className="flex gap-3 p-6 border-t border-slate-200 dark:border-slate-700">
          <button
            onClick={onClose}
            disabled={isPending}
            className="flex-1 px-4 py-2 border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-300 font-medium rounded-lg hover:bg-slate-50 dark:hover:bg-slate-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            关闭
          </button>
          {!feedback && (
            <button
              onClick={handleRedeem}
              disabled={isPending || !code.trim()}
              className="flex-1 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white font-medium rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center justify-center gap-2"
            >
              {isPending && <Loader2 className="w-4 h-4 animate-spin" />}
              {isPending ? '兑换中...' : '兑换'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

