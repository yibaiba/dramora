import { ArrowUp, ArrowDown } from 'lucide-react'
import type { WalletTransaction } from '../../api/types'

interface TransactionHistoryTableProps {
  transactions: WalletTransaction[]
  isLoading?: boolean
}

export default function TransactionHistoryTable({ transactions, isLoading }: TransactionHistoryTableProps) {
  const getTypeLabel = (kind: string) => {
    const labels: Record<string, string> = {
      credit: 'Credit',
      debit: 'Debit',
      refund: 'Refund',
      adjust: 'Adjustment',
    }
    return labels[kind] || kind
  }

  const getTypeColor = (kind: string) => {
    switch (kind) {
      case 'credit':
      case 'refund':
        return 'text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/20'
      case 'debit':
      case 'adjust':
        return 'text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20'
      default:
        return 'text-slate-600 dark:text-slate-400 bg-slate-50 dark:bg-slate-800'
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-slate-500 dark:text-slate-400">Loading transactions...</div>
      </div>
    )
  }

  if (!transactions || transactions.length === 0) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-slate-500 dark:text-slate-400">No transactions found</div>
      </div>
    )
  }

  return (
    <div className="overflow-x-auto rounded-lg border border-slate-200 dark:border-slate-700">
      <table className="w-full">
        <thead className="bg-slate-50 dark:bg-slate-800">
          <tr className="border-b border-slate-200 dark:border-slate-700">
            <th className="text-left px-4 py-3 text-sm font-semibold text-slate-700 dark:text-slate-300">Date</th>
            <th className="text-left px-4 py-3 text-sm font-semibold text-slate-700 dark:text-slate-300">Type</th>
            <th className="text-right px-4 py-3 text-sm font-semibold text-slate-700 dark:text-slate-300">Amount</th>
            <th className="text-right px-4 py-3 text-sm font-semibold text-slate-700 dark:text-slate-300">Balance</th>
            <th className="text-left px-4 py-3 text-sm font-semibold text-slate-700 dark:text-slate-300">Description</th>
          </tr>
        </thead>
        <tbody>
          {transactions.map((tx, idx) => (
            <tr
              key={tx.id}
              className={`border-b border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors ${
                idx % 2 === 0 ? 'bg-white dark:bg-slate-900' : 'bg-slate-50 dark:bg-slate-900/50'
              }`}
            >
              <td className="px-4 py-3 text-sm text-slate-900 dark:text-white">
                {new Date(tx.created_at).toLocaleDateString()} {new Date(tx.created_at).toLocaleTimeString()}
              </td>
              <td className="px-4 py-3">
                <span className={`inline-flex items-center gap-1 px-2 py-1 rounded text-sm font-medium ${getTypeColor(tx.kind)}`}>
                  {tx.direction > 0 ? <ArrowUp className="w-3 h-3" /> : <ArrowDown className="w-3 h-3" />}
                  {getTypeLabel(tx.kind)}
                </span>
              </td>
              <td className="px-4 py-3 text-sm text-right font-semibold text-slate-900 dark:text-white">
                {tx.direction > 0 ? '+' : '-'}{Math.abs(tx.amount)} 积分
              </td>
              <td className="px-4 py-3 text-sm text-right text-slate-600 dark:text-slate-400">{tx.balance_after}</td>
              <td className="px-4 py-3 text-sm text-slate-600 dark:text-slate-400">
                <div title={tx.reason} className="truncate max-w-xs">
                  {tx.reason || '-'}
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
