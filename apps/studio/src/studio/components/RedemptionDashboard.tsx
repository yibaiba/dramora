import { Loader2 } from 'lucide-react'
import { useGetCampaignStats } from '../../api/hooks'

interface RedemptionDashboardProps {
  campaignId?: string
}

export default function RedemptionDashboard({ campaignId }: RedemptionDashboardProps) {
  const { data: stats, isLoading } = useGetCampaignStats(campaignId)

  if (!campaignId) {
    return (
      <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-6">
        <p className="text-slate-500 dark:text-slate-400">请选择一个活动来查看统计信息</p>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-6 flex items-center justify-center h-64">
        <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
      </div>
    )
  }

  if (!stats) {
    return (
      <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-6">
        <p className="text-slate-500 dark:text-slate-400">无法加载统计信息</p>
      </div>
    )
  }

  const redemptionRate =
    stats.totalCodes > 0 ? ((stats.redeemedCodes / stats.totalCodes) * 100).toFixed(1) : '0'

  const statCards = [
    {
      title: '已生成码',
      value: stats.totalCodes,
      subtitle: '总数',
    },
    {
      title: '已兑现码',
      value: stats.redeemedCodes,
      subtitle: `${redemptionRate}% 兑换率`,
    },
    {
      title: '总发放金额',
      value: stats.totalAmount,
      subtitle: '积分',
    },
    {
      title: '已兑现金额',
      value: stats.redeemedAmount,
      subtitle: '积分',
    },
  ]

  return (
    <div className="space-y-6">
      {/* Progress Bar */}
      <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white">兑换进度</h3>
          <span className="text-sm font-medium text-slate-600 dark:text-slate-400">
            {stats.redeemedCodes} / {stats.totalCodes}
          </span>
        </div>
        <div className="w-full bg-slate-200 dark:bg-slate-700 rounded-full h-3 overflow-hidden">
          <div
            className="bg-blue-500 h-full transition-all duration-300"
            style={{ width: `${redemptionRate}%` }}
          />
        </div>
        <div className="mt-2 text-sm text-slate-600 dark:text-slate-400">
          {redemptionRate}% 已兑换
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {statCards.map((card, idx) => (
          <div
            key={idx}
            className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-4"
          >
            <p className="text-sm font-medium text-slate-600 dark:text-slate-400">{card.title}</p>
            <p className="text-2xl font-bold text-slate-900 dark:text-white mt-2">{card.value}</p>
            <p className="text-xs text-slate-500 dark:text-slate-500 mt-1">{card.subtitle}</p>
          </div>
        ))}
      </div>

      {/* Details Section */}
      <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-6">
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">统计详情</h3>
        <div className="space-y-3">
          <div className="flex items-center justify-between py-2 border-b border-slate-200 dark:border-slate-700">
            <span className="text-slate-700 dark:text-slate-300">总发放金额</span>
            <span className="font-semibold text-slate-900 dark:text-white">{stats.totalAmount} 积分</span>
          </div>
          <div className="flex items-center justify-between py-2 border-b border-slate-200 dark:border-slate-700">
            <span className="text-slate-700 dark:text-slate-300">已兑现金额</span>
            <span className="font-semibold text-slate-900 dark:text-white">{stats.redeemedAmount} 积分</span>
          </div>
          <div className="flex items-center justify-between py-2 border-b border-slate-200 dark:border-slate-700">
            <span className="text-slate-700 dark:text-slate-300">未兑现金额</span>
            <span className="font-semibold text-slate-900 dark:text-white">
              {stats.totalAmount - stats.redeemedAmount} 积分
            </span>
          </div>
          <div className="flex items-center justify-between py-2">
            <span className="text-slate-700 dark:text-slate-300">平均兑换金额</span>
            <span className="font-semibold text-slate-900 dark:text-white">
              {stats.redeemedCodes > 0 ? (stats.redeemedAmount / stats.redeemedCodes).toFixed(0) : '0'} 积分
            </span>
          </div>
        </div>
      </div>

      {/* Generated At */}
      <div className="text-xs text-slate-500 dark:text-slate-400 text-right">
        数据最后更新于：{new Date(stats.generatedAt).toLocaleString()}
      </div>
    </div>
  )
}
