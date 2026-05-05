import { useState } from 'react'
import { X, Loader2 } from 'lucide-react'
import type { CreateBatchSubmissionRequest } from '../../api/types'

interface CreateBatchSubmissionDialogProps {
  isOpen: boolean
  onClose: () => void
  onSubmit: (req: CreateBatchSubmissionRequest) => Promise<void>
  availableVideoIds: string[]
}

export default function CreateBatchSubmissionDialog({
  isOpen,
  onClose,
  onSubmit,
  availableVideoIds,
}: CreateBatchSubmissionDialogProps) {
  const [selectedVideoIds, setSelectedVideoIds] = useState<Set<string>>(new Set())
  const [concurrencyLimit, setConcurrencyLimit] = useState(4)
  const [retryLimit, setRetryLimit] = useState(2)
  const [isSubmitting, setIsSubmitting] = useState(false)

  const selectAllVideos = (selected: boolean) => {
    if (selected) {
      setSelectedVideoIds(new Set(availableVideoIds))
    } else {
      setSelectedVideoIds(new Set())
    }
  }

  const toggleVideo = (videoId: string) => {
    const newSet = new Set(selectedVideoIds)
    if (newSet.has(videoId)) {
      newSet.delete(videoId)
    } else {
      newSet.add(videoId)
    }
    setSelectedVideoIds(newSet)
  }

  const handleSubmit = async () => {
    if (selectedVideoIds.size === 0) {
      alert('Please select at least one video')
      return
    }

    setIsSubmitting(true)
    try {
      await onSubmit({
        videoIds: Array.from(selectedVideoIds),
        concurrencyLimit,
        retryLimit,
        parameters: {}, // Support for future parameter customization
      })
      // Reset form
      setSelectedVideoIds(new Set())
      setConcurrencyLimit(4)
      setRetryLimit(2)
      onClose()
    } catch (error) {
      console.error('Failed to create batch submission:', error)
      alert('Failed to create batch submission. Please try again.')
    } finally {
      setIsSubmitting(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-white dark:bg-slate-900 rounded-lg shadow-lg max-w-2xl w-full max-h-96 flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-700 px-6 py-4">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Create Batch Submission</h2>
          <button
            onClick={onClose}
            disabled={isSubmitting}
            className="p-1 hover:bg-slate-100 dark:hover:bg-slate-800 rounded disabled:opacity-50"
            aria-label="Close dialog"
          >
            <X size={20} />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-6 py-4 space-y-6">
          {/* Video Selection */}
          <div>
            <label className="block text-sm font-medium text-slate-900 dark:text-white mb-3">
              Select Videos ({selectedVideoIds.size} selected)
            </label>
            <div className="flex items-center gap-2 mb-3">
              <input
                type="checkbox"
                id="select-all"
                checked={selectedVideoIds.size === availableVideoIds.length && availableVideoIds.length > 0}
                onChange={(e) => selectAllVideos(e.target.checked)}
                className="w-4 h-4 rounded border-slate-300 dark:border-slate-600"
              />
              <label htmlFor="select-all" className="text-sm text-slate-700 dark:text-slate-300">
                Select all
              </label>
            </div>
            <div className="space-y-2 max-h-48 overflow-y-auto border border-slate-200 dark:border-slate-700 rounded p-3 bg-slate-50 dark:bg-slate-800/50">
              {availableVideoIds.length > 0 ? (
                availableVideoIds.map((videoId) => (
                  <label key={videoId} className="flex items-center gap-2 p-2 hover:bg-slate-100 dark:hover:bg-slate-700 rounded cursor-pointer">
                    <input
                      type="checkbox"
                      checked={selectedVideoIds.has(videoId)}
                      onChange={() => toggleVideo(videoId)}
                      className="w-4 h-4 rounded border-slate-300 dark:border-slate-600"
                    />
                    <span className="text-sm text-slate-700 dark:text-slate-300 font-mono">{videoId.slice(0, 12)}...</span>
                  </label>
                ))
              ) : (
                <div className="text-sm text-slate-500 dark:text-slate-400 p-2">No videos available</div>
              )}
            </div>
          </div>

          {/* Settings */}
          <div className="grid grid-cols-2 gap-4">
            {/* Concurrency Limit */}
            <div>
              <label htmlFor="concurrency" className="block text-sm font-medium text-slate-900 dark:text-white mb-2">
                Concurrency Limit (1-8)
              </label>
              <input
                type="number"
                id="concurrency"
                min={1}
                max={8}
                value={concurrencyLimit}
                onChange={(e) => setConcurrencyLimit(Math.min(8, Math.max(1, parseInt(e.target.value) || 1)))}
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
              />
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">How many videos to generate in parallel</p>
            </div>

            {/* Retry Limit */}
            <div>
              <label htmlFor="retry" className="block text-sm font-medium text-slate-900 dark:text-white mb-2">
                Retry Limit (0-5)
              </label>
              <input
                type="number"
                id="retry"
                min={0}
                max={5}
                value={retryLimit}
                onChange={(e) => setRetryLimit(Math.min(5, Math.max(0, parseInt(e.target.value) || 0)))}
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-md bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
              />
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">Failed videos will retry this many times</p>
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="border-t border-slate-200 dark:border-slate-700 px-6 py-4 flex items-center justify-end gap-3">
          <button
            onClick={onClose}
            disabled={isSubmitting}
            className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-md disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={handleSubmit}
            disabled={isSubmitting || selectedVideoIds.size === 0}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          >
            {isSubmitting && <Loader2 size={16} className="animate-spin" />}
            {isSubmitting ? 'Creating...' : 'Create Batch'}
          </button>
        </div>
      </div>
    </div>
  )
}
