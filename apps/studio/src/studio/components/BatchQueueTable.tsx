import { useState } from 'react'
import { useReactTable, getCoreRowModel, getPaginationRowModel, flexRender, type ColumnDef } from '@tanstack/react-table'
import { X, RotateCcw, ChevronLeft, ChevronRight } from 'lucide-react'
import type { BatchSubmission } from '../../api/types'

interface BatchQueueTableProps {
  batches: BatchSubmission[]
  isLoading?: boolean
  onCancel?: (batchId: string) => void
  onRetry?: (batchId: string) => void
}

const getStatusBadge = (status: BatchSubmission['status']) => {
  const baseClasses = 'inline-flex items-center gap-1 px-2 py-1 rounded-full text-sm font-medium'
  switch (status) {
    case 'pending':
      return `${baseClasses} bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300`
    case 'queued':
      return `${baseClasses} bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300`
    case 'processing':
      return `${baseClasses} bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300`
    case 'completed':
      return `${baseClasses} bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300`
    case 'cancelled':
      return `${baseClasses} bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300`
    default:
      return baseClasses
  }
}

const getProgressPercentage = (batch: BatchSubmission): number => {
  if (batch.videoCount === 0) return 0
  return Math.round(((batch.completedCount + batch.failedCount + batch.cancelledCount) / batch.videoCount) * 100)
}

export default function BatchQueueTable({ batches, isLoading, onCancel, onRetry }: BatchQueueTableProps) {
  const [pageIndex, setPageIndex] = useState(0)
  const pageSize = 10

  const columns: ColumnDef<BatchSubmission>[] = [
    {
      accessorKey: 'id',
      header: 'Batch ID',
      cell: (info) => (
        <span className="font-mono text-sm text-slate-600 dark:text-slate-400">
          {(info.getValue() as string).slice(0, 8)}
        </span>
      ),
    },
    {
      accessorKey: 'status',
      header: 'Status',
      cell: (info) => {
        const status = info.getValue() as BatchSubmission['status']
        return <span className={getStatusBadge(status)}>{status}</span>
      },
    },
    {
      accessorKey: 'videoCount',
      header: 'Videos',
      cell: (info) => <span className="text-center">{info.getValue() as number}</span>,
    },
    {
      header: 'Progress',
      accessorFn: (row) => row,
      cell: (info) => {
        const batch = info.row.original
        const progress = getProgressPercentage(batch)
        return (
          <div className="w-32">
            <div className="flex items-center gap-2">
              <div className="flex-1 h-2 bg-slate-200 dark:bg-slate-700 rounded-full overflow-hidden">
                <div className="h-full bg-blue-500 transition-all" style={{ width: `${progress}%` }} />
              </div>
              <span className="text-sm text-slate-600 dark:text-slate-400">{progress}%</span>
            </div>
            <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              {batch.completedCount + batch.failedCount + batch.cancelledCount}/{batch.videoCount}
            </div>
          </div>
        )
      },
    },
    {
      accessorKey: 'concurrencyLimit',
      header: 'Concurrency',
      cell: (info) => <span className="text-center">{info.getValue() as number}</span>,
    },
    {
      accessorKey: 'createdAt',
      header: 'Created',
      cell: (info) => {
        const date = new Date(info.getValue() as string)
        return (
          <span className="text-sm text-slate-600 dark:text-slate-400">
            {date.toLocaleDateString()} {date.toLocaleTimeString()}
          </span>
        )
      },
    },
    {
      id: 'actions',
      header: 'Actions',
      cell: (info) => {
        const batch = info.row.original
        const canCancel = batch.status !== 'completed' && batch.status !== 'cancelled'
        const canRetry = batch.failedCount > 0

        return (
          <div className="flex items-center gap-2 justify-end">
            {canCancel && (
              <button
                onClick={() => onCancel?.(batch.id)}
                className="p-1.5 hover:bg-red-100 dark:hover:bg-red-900/30 rounded transition-colors text-red-600 dark:text-red-400"
                title="Cancel batch"
                aria-label="Cancel batch"
              >
                <X size={16} />
              </button>
            )}
            {canRetry && (
              <button
                onClick={() => onRetry?.(batch.id)}
                className="p-1.5 hover:bg-amber-100 dark:hover:bg-amber-900/30 rounded transition-colors text-amber-600 dark:text-amber-400"
                title="Retry failed videos"
                aria-label="Retry failed videos"
              >
                <RotateCcw size={16} />
              </button>
            )}
          </div>
        )
      },
    },
  ]

  const table = useReactTable({
    data: batches,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    state: {
      pagination: {
        pageIndex,
        pageSize,
      },
    },
    onPaginationChange: (updater) => {
      const newPageState = typeof updater === 'function' ? updater({ pageIndex, pageSize }) : updater
      setPageIndex(newPageState.pageIndex)
    },
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64 border border-slate-200 dark:border-slate-700 rounded-lg">
        <div className="text-slate-500 dark:text-slate-400">Loading batches...</div>
      </div>
    )
  }

  if (!batches || batches.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 border border-slate-200 dark:border-slate-700 rounded-lg">
        <div className="text-slate-500 dark:text-slate-400">No batch submissions yet</div>
      </div>
    )
  }

  const { rows } = table.getRowModel()
  const pageCount = table.getPageCount()

  return (
    <div className="space-y-4">
      <div className="overflow-x-auto rounded-lg border border-slate-200 dark:border-slate-700">
        <table className="w-full">
          <thead className="bg-slate-50 dark:bg-slate-800">
            <tr className="border-b border-slate-200 dark:border-slate-700">
              {table.getHeaderGroups()[0]?.headers.map((header) => (
                <th
                  key={header.id}
                  className="text-left px-4 py-3 text-sm font-semibold text-slate-700 dark:text-slate-300"
                >
                  {header.isPlaceholder ? null : flexRender(header.column.columnDef.header, header.getContext())}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, idx) => (
              <tr
                key={row.id}
                className={`border-b border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors ${
                  idx % 2 === 0 ? 'bg-white dark:bg-slate-900' : 'bg-slate-50 dark:bg-slate-900/50'
                }`}
              >
                {row.getVisibleCells().map((cell) => (
                  <td key={cell.id} className="px-4 py-3 text-sm">
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      <div className="flex items-center justify-between">
        <div className="text-sm text-slate-600 dark:text-slate-400">
          Page {pageIndex + 1} of {pageCount} (Total: {batches.length} batches)
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => table.previousPage()}
            disabled={!table.getCanPreviousPage()}
            className="p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            aria-label="Previous page"
          >
            <ChevronLeft size={18} />
          </button>
          <button
            onClick={() => table.nextPage()}
            disabled={!table.getCanNextPage()}
            className="p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            aria-label="Next page"
          >
            <ChevronRight size={18} />
          </button>
        </div>
      </div>
    </div>
  )
}
