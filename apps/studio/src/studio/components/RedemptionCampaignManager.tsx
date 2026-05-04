import { useState } from 'react'
import { Loader2, Download, Copy, Check } from 'lucide-react'
import { useGenerateRedemptionCodes } from '../../api/hooks'

interface ApiError {
  response?: {
    data?: {
      message?: string
    }
  }
}

export default function RedemptionCampaignManager() {
  const [count, setCount] = useState('100')
  const [amount, setAmount] = useState('1000')
  const [expiresAt, setExpiresAt] = useState('')
  const [reason, setReason] = useState('')
  const [generatedCodes, setGeneratedCodes] = useState<string[]>([])
  const [copySuccess, setCopySuccess] = useState(false)

  const { mutate: generate, isPending } = useGenerateRedemptionCodes()

  const handleGenerate = () => {
    const countNum = parseInt(count)
    const amountNum = parseInt(amount)

    if (!countNum || countNum <= 0) {
      alert('请输入有效的数量')
      return
    }
    if (!amountNum || amountNum <= 0) {
      alert('请输入有效的金额')
      return
    }

    generate(
      {
        count: countNum,
        amount: amountNum,
        expiresAt: expiresAt || undefined,
        reason: reason || undefined,
      },
      {
        onSuccess: (response) => {
          setGeneratedCodes(response.codes)
        },
        onError: (error) => {
          const apiError = error as ApiError
          const message =
            apiError?.response?.data?.message || '生成赎回码失败'
          alert(message)
        },
      }
    )
  }

  const downloadCSV = () => {
    if (generatedCodes.length === 0) return

    const csv = [
      'Code,Amount,Status',
      ...generatedCodes.map((code) => `${code},${amount},unused`),
    ].join('\n')

    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
    const link = document.createElement('a')
    link.href = URL.createObjectURL(blob)
    link.download = `redemption-codes-${new Date().toISOString().split('T')[0]}.csv`
    link.click()
  }

  const copyToClipboard = () => {
    const text = generatedCodes.join('\n')
    navigator.clipboard.writeText(text)
    setCopySuccess(true)
    setTimeout(() => setCopySuccess(false), 2000)
  }

  return (
    <div className="w-full max-w-4xl mx-auto">
      <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 overflow-hidden shadow-sm">
        {/* Header */}
        <div className="p-6 border-b border-slate-200 dark:border-slate-700">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-white">生成赎回码</h2>
          <p className="text-sm text-slate-600 dark:text-slate-400 mt-1">
            批量生成赎回码供用户兑换
          </p>
        </div>

        {/* Content */}
        <div className="p-6">
          {generatedCodes.length === 0 ? (
            <div className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    生成数量
                  </label>
                  <input
                    type="number"
                    value={count}
                    onChange={(e) => setCount(e.target.value)}
                    min="1"
                    max="10000"
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                    单个金额（积分）
                  </label>
                  <input
                    type="number"
                    value={amount}
                    onChange={(e) => setAmount(e.target.value)}
                    min="1"
                    className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  过期时间（可选）
                </label>
                <input
                  type="datetime-local"
                  value={expiresAt}
                  onChange={(e) => setExpiresAt(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  备注/原因（可选）
                </label>
                <input
                  type="text"
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  placeholder="例如：月度奖励、合作方赠送"
                  className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
                <p className="text-sm text-blue-700 dark:text-blue-300">
                  总金额：{parseInt(count || '0') * parseInt(amount || '0')} 积分
                </p>
              </div>

              <button
                onClick={handleGenerate}
                disabled={isPending}
                className="w-full px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white font-medium rounded-lg disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center justify-center gap-2"
              >
                {isPending && <Loader2 className="w-4 h-4 animate-spin" />}
                {isPending ? '生成中...' : '生成赎回码'}
              </button>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
                <p className="text-sm font-semibold text-green-900 dark:text-green-200">
                  ✓ 成功生成 {generatedCodes.length} 个赎回码
                </p>
                <p className="text-sm text-green-700 dark:text-green-300 mt-1">
                  总金额：{generatedCodes.length * parseInt(amount)} 积分
                </p>
              </div>

              <div className="max-h-48 overflow-y-auto bg-slate-50 dark:bg-slate-800 rounded-lg p-4">
                <div className="space-y-2">
                  {generatedCodes.map((code, idx) => (
                    <div
                      key={idx}
                      className="text-sm font-mono text-slate-900 dark:text-white bg-white dark:bg-slate-700 px-3 py-2 rounded"
                    >
                      {code}
                    </div>
                  ))}
                </div>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <button
                  onClick={copyToClipboard}
                  className="px-4 py-2 bg-purple-500 hover:bg-purple-600 text-white font-medium rounded-lg transition-colors flex items-center justify-center gap-2"
                >
                  {copySuccess ? (
                    <>
                      <Check className="w-4 h-4" />
                      已复制
                    </>
                  ) : (
                    <>
                      <Copy className="w-4 h-4" />
                      复制到剪贴板
                    </>
                  )}
                </button>
                <button
                  onClick={downloadCSV}
                  className="px-4 py-2 bg-emerald-500 hover:bg-emerald-600 text-white font-medium rounded-lg transition-colors flex items-center justify-center gap-2"
                >
                  <Download className="w-4 h-4" />
                  下载 CSV
                </button>
              </div>

              <button
                onClick={() => {
                  setGeneratedCodes([])
                  setCount('100')
                  setAmount('1000')
                  setExpiresAt('')
                  setReason('')
                }}
                className="w-full px-4 py-2 border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-300 font-medium rounded-lg hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors"
              >
                生成新的赎回码
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
