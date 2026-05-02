import React, { useState } from 'react'
import { useAdminBillingReports, useGenerateAdminBillingReport, useAdminBillingReportSummary } from '../../api/hooks'
import type { GenerateBillingReportRequest, BillingReport } from '../../api/types'
import { AlertCircle, Loader2, Plus, TrendingDown, TrendingUp, DollarSign } from 'lucide-react'

export function BillingReportsPage() {
  const [offset, setOffset] = useState(0)
  const [showGenerateDialog, setShowGenerateDialog] = useState(false)
  const [selectedReport, setSelectedReport] = useState<string | null>(null)

  // 列表查询
  const limit = 20
  const { data, isLoading, error } = useAdminBillingReports({ limit, offset }, true)
  const reports = data?.reports ?? []
  const total = data?.total ?? 0

  // 生成报表 mutation
  const generateMutation = useGenerateAdminBillingReport()

  const handleGenerateReport = (periodStart: number, periodEnd: number) => {
    const req: GenerateBillingReportRequest = { period_start: periodStart, period_end: periodEnd }
    generateMutation.mutate(req, {
      onSuccess: () => {
        setShowGenerateDialog(false)
      },
    })
  }

  const formatDate = (timestamp: number) => {
    return new Date(timestamp * 1000).toLocaleDateString('zh-CN')
  }

  const formatAmount = (amount: number) => {
    return amount.toLocaleString()
  }

  const getStatusBadge = (status: string) => {
    return status === 'draft'
      ? 'px-3 py-1 rounded-full text-xs font-medium bg-slate-200 dark:bg-slate-700 text-slate-900 dark:text-slate-100'
      : 'px-3 py-1 rounded-full text-xs font-medium bg-green-200 dark:bg-green-900 text-green-900 dark:text-green-100'
  }

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-slate-900 dark:text-white">计费清算报表</h1>
          <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
            查看和管理组织的财务清算报表
          </p>
        </div>
        <button
          onClick={() => setShowGenerateDialog(true)}
          className="inline-flex items-center gap-2 rounded-lg bg-blue-600 hover:bg-blue-700 px-4 py-2 font-medium text-white transition-colors"
        >
          <Plus className="h-4 w-4" />
          生成新报表
        </button>
      </div>

      {/* 生成报表对话框 */}
      {showGenerateDialog && (
        <GenerateReportDialog
          onClose={() => setShowGenerateDialog(false)}
          onGenerate={handleGenerateReport}
          isLoading={generateMutation.isPending}
          error={generateMutation.error?.message}
        />
      )}

      {/* 错误显示 */}
      {error && (
        <div className="flex items-start gap-3 rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-900 dark:bg-red-900/20">
          <AlertCircle className="h-5 w-5 flex-shrink-0 text-red-600 dark:text-red-400 mt-0.5" />
          <div>
            <p className="font-medium text-red-900 dark:text-red-200">加载失败</p>
            <p className="text-sm text-red-800 dark:text-red-300">{error.message}</p>
          </div>
        </div>
      )}

      {/* 报表表格 */}
      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-blue-600" />
        </div>
      ) : reports.length === 0 ? (
        <div className="rounded-lg border border-slate-200 bg-slate-50 p-8 text-center dark:border-slate-700 dark:bg-slate-800">
          <DollarSign className="mx-auto h-12 w-12 text-slate-400 dark:text-slate-600" />
          <p className="mt-2 text-slate-600 dark:text-slate-400">暂无报表</p>
        </div>
      ) : (
        <>
          <div className="overflow-x-auto rounded-lg border border-slate-200 dark:border-slate-700">
            <table className="w-full text-sm">
              <thead className="border-b border-slate-200 bg-slate-50 dark:border-slate-700 dark:bg-slate-800">
                <tr>
                  <th className="px-6 py-3 text-left font-semibold text-slate-900 dark:text-white">
                    周期
                  </th>
                  <th className="px-6 py-3 text-left font-semibold text-slate-900 dark:text-white">
                    总支出
                  </th>
                  <th className="px-6 py-3 text-left font-semibold text-slate-900 dark:text-white">
                    总收入
                  </th>
                  <th className="px-6 py-3 text-left font-semibold text-slate-900 dark:text-white">
                    净额
                  </th>
                  <th className="px-6 py-3 text-left font-semibold text-slate-900 dark:text-white">
                    待结算
                  </th>
                  <th className="px-6 py-3 text-left font-semibold text-slate-900 dark:text-white">
                    状态
                  </th>
                  <th className="px-6 py-3 text-left font-semibold text-slate-900 dark:text-white">
                    生成时间
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                {reports.map((report: BillingReport) => (
                  <tr
                    key={report.id}
                    onClick={() => setSelectedReport(report.id)}
                    className="hover:bg-slate-50 cursor-pointer transition-colors dark:hover:bg-slate-700/50"
                  >
                    <td className="px-6 py-4 text-slate-900 dark:text-white">
                      {formatDate(report.period_start)} ~ {formatDate(report.period_end)}
                    </td>
                    <td className="px-6 py-4">
                      <span className="inline-flex items-center gap-1 text-red-600 dark:text-red-400">
                        <TrendingDown className="h-3 w-3" />
                        {formatAmount(report.total_debit_amount)}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <span className="inline-flex items-center gap-1 text-green-600 dark:text-green-400">
                        <TrendingUp className="h-3 w-3" />
                        {formatAmount(report.total_credit_amount)}
                      </span>
                    </td>
                    <td className="px-6 py-4 font-medium text-slate-900 dark:text-white">
                      {formatAmount(Math.abs(report.net_amount))}
                    </td>
                    <td className="px-6 py-4 text-slate-600 dark:text-slate-400">
                      {report.pending_billing_count} ({formatAmount(report.pending_billing_amount)})
                    </td>
                    <td className="px-6 py-4">
                      <span className={getStatusBadge(report.status)}>
                        {report.status === 'draft' ? '草稿' : '已确认'}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-slate-500 dark:text-slate-400">
                      {formatDate(report.generated_at)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* 分页 */}
          <div className="flex items-center justify-between">
            <p className="text-sm text-slate-600 dark:text-slate-400">
              显示 {offset + 1}-{Math.min(offset + limit, total)} / 共 {total} 条
            </p>
            <div className="flex gap-2">
              <button
                disabled={offset === 0}
                onClick={() => setOffset(Math.max(0, offset - limit))}
                className="rounded-lg border border-slate-300 px-3 py-1 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50 disabled:cursor-not-allowed dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
              >
                上一页
              </button>
              <button
                disabled={offset + limit >= total}
                onClick={() => setOffset(offset + limit)}
                className="rounded-lg border border-slate-300 px-3 py-1 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50 disabled:cursor-not-allowed dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
              >
                下一页
              </button>
            </div>
          </div>
        </>
      )}

      {/* 详情侧边栏 */}
      {selectedReport && (
        <BillingReportDetailPanel
          reportID={selectedReport}
          onClose={() => setSelectedReport(null)}
        />
      )}
    </div>
  )
}

interface GenerateReportDialogProps {
  onClose: () => void
  onGenerate: (periodStart: number, periodEnd: number) => void
  isLoading: boolean
  error?: string
}

function GenerateReportDialog({ onClose, onGenerate, isLoading, error }: GenerateReportDialogProps) {
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!startDate || !endDate) return

    const start = new Date(startDate).getTime() / 1000
    const end = new Date(endDate).getTime() / 1000

    if (start >= end) {
      alert('开始日期必须早于结束日期')
      return
    }

    onGenerate(start, end)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-end bg-black/50 sm:items-center">
      <div className="relative w-full max-w-md rounded-t-lg bg-white p-6 dark:bg-slate-900 sm:rounded-lg">
        <button
          onClick={onClose}
          className="absolute right-4 top-4 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
        >
          ✕
        </button>

        <h2 className="text-xl font-bold text-slate-900 dark:text-white">生成清算报表</h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
          选择要清算的时间段（最多90天）
        </p>

        {error && (
          <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-600 dark:border-red-900 dark:bg-red-900/20 dark:text-red-400">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="mt-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
              开始日期
            </label>
            <input
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              required
              className="mt-2 w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-slate-600 dark:bg-slate-800 dark:text-white"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300">
              结束日期
            </label>
            <input
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              required
              className="mt-2 w-full rounded-lg border border-slate-300 px-3 py-2 dark:border-slate-600 dark:bg-slate-800 dark:text-white"
            />
          </div>

          <div className="flex gap-3 pt-4">
            <button
              type="button"
              onClick={onClose}
              disabled={isLoading}
              className="flex-1 rounded-lg border border-slate-300 px-4 py-2 font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
            >
              取消
            </button>
            <button
              type="submit"
              disabled={isLoading}
              className="flex-1 inline-flex items-center justify-center gap-2 rounded-lg bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 px-4 py-2 font-medium text-white transition-colors"
            >
              {isLoading && <Loader2 className="h-4 w-4 animate-spin" />}
              生成报表
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

interface BillingReportDetailPanelProps {
  reportID: string
  onClose: () => void
}

function BillingReportDetailPanel({ reportID, onClose }: BillingReportDetailPanelProps) {
  const { data, isLoading } = useAdminBillingReportSummary(reportID, true)

  if (isLoading) {
    return (
      <div className="fixed inset-y-0 right-0 w-96 border-l border-slate-200 bg-white p-6 dark:border-slate-700 dark:bg-slate-900 flex items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-blue-600" />
      </div>
    )
  }

  if (!data) {
    return null
  }

  return (
    <div className="fixed inset-y-0 right-0 w-96 border-l border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-900 overflow-y-auto">
      <div className="p-6 space-y-6">
        <div className="flex items-center justify-between">
          <h3 className="text-lg font-bold text-slate-900 dark:text-white">报表详情</h3>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
          >
            ✕
          </button>
        </div>

        {/* 统计卡片 */}
        <div className="space-y-4">
          <div className="rounded-lg bg-slate-50 p-4 dark:bg-slate-800">
            <p className="text-xs font-medium text-slate-600 dark:text-slate-400">总支出</p>
            <p className="mt-1 text-xl font-bold text-red-600 dark:text-red-400">
              {data.total_debit_amount.toLocaleString()}
            </p>
          </div>

          <div className="rounded-lg bg-slate-50 p-4 dark:bg-slate-800">
            <p className="text-xs font-medium text-slate-600 dark:text-slate-400">总收入</p>
            <p className="mt-1 text-xl font-bold text-green-600 dark:text-green-400">
              {data.total_credit_amount.toLocaleString()}
            </p>
          </div>

          <div className="rounded-lg bg-slate-50 p-4 dark:bg-slate-800">
            <p className="text-xs font-medium text-slate-600 dark:text-slate-400">净额</p>
            <p className="mt-1 text-xl font-bold text-slate-900 dark:text-white">
              {data.net_amount.toLocaleString()}
            </p>
          </div>
        </div>

        {/* 待结算状态 */}
        <div className="space-y-2 rounded-lg border border-slate-200 p-4 dark:border-slate-700">
          <p className="font-medium text-slate-900 dark:text-white">待结算状态</p>
          <div className="text-sm text-slate-600 dark:text-slate-400 space-y-1">
            <div className="flex justify-between">
              <span>待处理：{data.pending_billing_count}</span>
            </div>
            <div className="flex justify-between">
              <span>已解决：{data.resolved_billing_count}</span>
            </div>
            <div className="flex justify-between">
              <span>失败：{data.failed_billing_count}</span>
            </div>
          </div>
        </div>

        <button
          onClick={onClose}
          className="w-full rounded-lg bg-slate-200 hover:bg-slate-300 dark:bg-slate-700 dark:hover:bg-slate-600 px-4 py-2 font-medium text-slate-900 dark:text-white transition-colors"
        >
          关闭
        </button>
      </div>
    </div>
  )
}
