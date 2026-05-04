import { useState } from 'react'
import { Loader2, X, Mail } from 'lucide-react'

interface EmailDistributionDialogProps {
  isOpen: boolean
  onClose: () => void
  codes: string[]
  amount: number
  campaignName?: string
  campaignId?: string
}

export default function EmailDistributionDialog({
  isOpen,
  onClose,
  codes,
  amount,
  campaignName = '',
  campaignId = '',
}: EmailDistributionDialogProps) {
  const [recipients, setRecipients] = useState('')
  const [subject, setSubject] = useState('🎁 您的赎回码已准备好')
  const [isSending, setIsSending] = useState(false)

  if (!isOpen) return null

  const recipientList = recipients
    .split(/[,\n]/)
    .map((r) => r.trim())
    .filter((r) => r.length > 0)

  const handleSendEmail = async () => {
    if (recipientList.length === 0) {
      alert('请输入至少一个收件人邮箱')
      return
    }

    if (codes.length === 0) {
      alert('没有可以分发的赎回码')
      return
    }

    setIsSending(true)
    try {
      const response = await fetch('/api/v1/admin/redemption-codes:send-email', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          codes,
          recipients: recipientList,
          subject,
          campaign_name: campaignName,
          campaign_id: campaignId,
        }),
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.message || '发送失败')
      }

      const data = await response.json()
      alert(`邮件分发任务已投入队列，任务ID: ${data.task_id}`)
      onClose()
    } catch (error) {
      alert(`发送失败: ${error instanceof Error ? error.message : '未知错误'}`)
    } finally {
      setIsSending(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 bg-black bg-opacity-50 flex items-center justify-center p-4">
      <div className="bg-white dark:bg-slate-900 rounded-lg shadow-lg max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between p-6 border-b border-slate-200 dark:border-slate-700 sticky top-0 bg-white dark:bg-slate-900">
          <div className="flex items-center gap-2">
            <Mail className="w-5 h-5 text-slate-600 dark:text-slate-400" />
            <h2 className="text-lg font-semibold text-slate-900 dark:text-white">邮件分发</h2>
          </div>
          <button
            onClick={onClose}
            className="text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              收件人邮箱 <span className="text-red-500">*</span>
            </label>
            <textarea
              value={recipients}
              onChange={(e) => setRecipients(e.target.value)}
              placeholder="支持逗号或换行分隔&#10;例如: user1@example.com, user2@example.com"
              className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
              rows={4}
            />
            <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
              {recipientList.length} 个收件人
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              邮件主题 <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              placeholder="例如: 🎁 您的赎回码已准备好"
              className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div className="p-4 bg-blue-50 dark:bg-blue-900 border border-blue-200 dark:border-blue-800 rounded-md">
            <p className="text-sm text-blue-900 dark:text-blue-100">
              <strong>分发信息:</strong>
              <br />
              - 赎回码数量: {codes.length}
              <br />
              - 每个收件人将获得1个赎回码
              <br />
              - 积分金额: {amount}
            </p>
          </div>
        </div>

        <div className="flex items-center justify-end gap-3 p-6 border-t border-slate-200 dark:border-slate-700 sticky bottom-0 bg-white dark:bg-slate-900">
          <button
            onClick={onClose}
            disabled={isSending}
            className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-md disabled:opacity-50 disabled:cursor-not-allowed transition"
          >
            取消
          </button>
          <button
            onClick={handleSendEmail}
            disabled={isSending || recipientList.length === 0 || codes.length === 0}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md disabled:opacity-50 disabled:cursor-not-allowed transition flex items-center gap-2"
          >
            {isSending && <Loader2 className="w-4 h-4 animate-spin" />}
            {isSending ? '发送中...' : '发送邮件'}
          </button>
        </div>
      </div>
    </div>
  )
}
