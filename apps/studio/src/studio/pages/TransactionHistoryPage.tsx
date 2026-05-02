import { useState } from 'react'
import { Loader2, ChevronLeft, ChevronRight } from 'lucide-react'
import { useWalletTransactions } from '../../api/hooks'
import TransactionHistoryTable from '../components/TransactionHistoryTable'
import type { WalletKind } from '../../api/types'

export default function TransactionHistoryPage() {
  const [limit] = useState(50)
  const [offset, setOffset] = useState(0)
  const [selectedKind, setSelectedKind] = useState<WalletKind | ''>('')

  const params = { limit, offset, kind: selectedKind || undefined }
  const { data, isLoading } = useWalletTransactions(params)

  const transactions = data?.transactions ?? []
  const hasMore = data?.has_more ?? false

  const handleNext = () => {
    if (hasMore) {
      setOffset(offset + limit)
    }
  }

  const handlePrevious = () => {
    if (offset > 0) {
      setOffset(Math.max(0, offset - limit))
    }
  }

  return (
    <div className="w-full max-w-6xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-slate-900 dark:text-white mb-2">Transaction History</h1>
        <p className="text-slate-600 dark:text-slate-400">View all your wallet transactions and activity</p>
      </div>

      {/* Filters */}
      <div className="mb-6 p-4 bg-slate-50 dark:bg-slate-800 rounded-lg">
        <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-3">Filter by Type</label>
        <div className="flex gap-2 flex-wrap">
          {['', 'credit', 'debit', 'refund', 'adjust'].map((kind) => (
            <button
              key={kind}
              onClick={() => {
                setSelectedKind(kind as WalletKind | '')
                setOffset(0)
              }}
              className={`px-3 py-2 rounded text-sm font-medium transition-colors ${
                selectedKind === kind
                  ? 'bg-blue-500 text-white'
                  : 'bg-white dark:bg-slate-700 text-slate-700 dark:text-slate-300 border border-slate-300 dark:border-slate-600 hover:bg-slate-100 dark:hover:bg-slate-600'
              }`}
            >
              {kind ? kind.charAt(0).toUpperCase() + kind.slice(1) : 'All'}
            </button>
          ))}
        </div>
      </div>

      {/* Table */}
      {isLoading ? (
        <div className="flex items-center justify-center h-64">
          <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
        </div>
      ) : (
        <>
          <TransactionHistoryTable transactions={transactions} isLoading={isLoading} />

          {/* Pagination */}
          <div className="mt-6 flex items-center justify-between">
            <div className="text-sm text-slate-600 dark:text-slate-400">
              Showing {offset + 1} to {offset + transactions.length} of {offset + transactions.length}
              {hasMore && '+'}
            </div>
            <div className="flex gap-2">
              <button
                onClick={handlePrevious}
                disabled={offset === 0}
                className="flex items-center gap-2 px-4 py-2 border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                <ChevronLeft className="w-4 h-4" />
                Previous
              </button>
              <button
                onClick={handleNext}
                disabled={!hasMore}
                className="flex items-center gap-2 px-4 py-2 border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                Next
                <ChevronRight className="w-4 h-4" />
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
