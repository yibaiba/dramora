import { useState } from 'react'
import { AlertCircle, ChevronDown, Loader2, Save } from 'lucide-react'
import { useAdminOperationCosts, useUpdateAdminOperationCosts, useAdminOperationCostHistory } from '../../api/hooks'

export function OperationCostsAdminPage() {
  const { data, isLoading, isError, error } = useAdminOperationCosts(true)
  const updateMutation = useUpdateAdminOperationCosts()
  const [expandedType, setExpandedType] = useState<string | null>(null)
  const historyQuery = useAdminOperationCostHistory(expandedType || '', expandedType !== null)
  const [editingCosts, setEditingCosts] = useState<Record<string, number>>({})
  const [reason, setReason] = useState('')

  const costs = data?.operation_costs || []

  const handleEdit = (operationType: string, newCost: number) => {
    setEditingCosts((prev) => ({
      ...prev,
      [operationType]: newCost,
    }))
  }

  const handleSave = async () => {
    const updates = Object.entries(editingCosts)
      .filter(([_, newCost]) => {
        const existing = costs.find((c) => c.operation_type === _)
        return existing && existing.credits_cost !== newCost
      })
      .map(([operationType, credits_cost]) => ({
        operation_type: operationType,
        credits_cost,
      }))

    if (updates.length === 0) {
      alert('No changes to save')
      return
    }

    try {
      await updateMutation.mutateAsync({
        updates,
        reason: reason || undefined,
      })
      setEditingCosts({})
      setReason('')
      alert('Operation costs updated successfully')
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to update costs'
      alert(`Error: ${errorMsg}`)
    }
  }

  return (
    <div className="min-h-screen bg-slate-950 p-6">
      <div className="mx-auto max-w-4xl">
        <h1 className="text-3xl font-bold text-white mb-6">Operation Costs Management</h1>

        {isError && (
          <div className="mb-6 flex items-center gap-3 rounded-lg border border-red-600 bg-red-900/20 p-4 text-red-300">
            <AlertCircle size={20} />
            <span>{error instanceof Error ? error.message : 'Failed to load costs'}</span>
          </div>
        )}

        {isLoading ? (
          <div className="flex items-center justify-center gap-2 text-slate-300">
            <Loader2 size={20} className="animate-spin" />
            Loading operation costs...
          </div>
        ) : (
          <>
            {/* Reason input */}
            <div className="mb-6">
              <label className="block text-sm font-medium text-slate-300 mb-2">Change Reason (optional)</label>
              <input
                type="text"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="e.g., 'Q1 pricing adjustment'"
                className="w-full rounded-lg border border-slate-700 bg-slate-800 px-4 py-2 text-white placeholder-slate-500 focus:border-blue-500 focus:outline-none"
              />
            </div>

            {/* Costs Table */}
            <div className="space-y-3 mb-6">
              {costs.map((cost) => (
                <div
                  key={cost.operation_type}
                  className="rounded-lg border border-slate-700 bg-slate-900 overflow-hidden"
                >
                  <div className="flex items-center justify-between p-4 cursor-pointer hover:bg-slate-800 transition"
                    onClick={() => setExpandedType(expandedType === cost.operation_type ? null : cost.operation_type)}
                  >
                    <div className="flex-1">
                      <h3 className="font-medium text-white">{cost.operation_type}</h3>
                      <p className="text-sm text-slate-400">Org: {cost.organization_id.substring(0, 8)}...</p>
                    </div>
                    <div className="flex items-center gap-6">
                      <div className="flex items-center gap-2">
                        <input
                          type="number"
                          value={editingCosts[cost.operation_type] ?? cost.credits_cost}
                          onChange={(e) => handleEdit(cost.operation_type, parseInt(e.target.value) || 0)}
                          onClick={(e) => e.stopPropagation()}
                          className="w-20 rounded border border-slate-600 bg-slate-700 px-2 py-1 text-white text-sm focus:border-blue-500 focus:outline-none"
                        />
                        <span className="text-sm text-slate-400">credits</span>
                      </div>
                      <ChevronDown
                        size={20}
                        className={`text-slate-400 transition ${expandedType === cost.operation_type ? 'rotate-180' : ''}`}
                      />
                    </div>
                  </div>

                  {/* History Section */}
                  {expandedType === cost.operation_type && (
                    <div className="border-t border-slate-700 bg-slate-800/50 p-4">
                      {historyQuery.isLoading ? (
                        <div className="flex items-center gap-2 text-slate-400">
                          <Loader2 size={16} className="animate-spin" />
                          Loading history...
                        </div>
                      ) : historyQuery.data?.history && historyQuery.data.history.length > 0 ? (
                        <div className="space-y-2">
                          {historyQuery.data.history.map((h, idx) => (
                            <div key={idx} className="text-sm text-slate-400">
                              <div className="font-medium text-slate-200">
                                {h.old_cost !== null ? `${h.old_cost} → ${h.new_cost}` : `Initial: ${h.new_cost}`}
                              </div>
                              <div className="text-xs text-slate-500">
                                {new Date(h.changed_at * 1000).toLocaleString()} by {h.changed_by}
                                {h.reason && ` - ${h.reason}`}
                              </div>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <p className="text-sm text-slate-500">No history available</p>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>

            {/* Save Button */}
            <div className="flex gap-3">
              <button
                onClick={handleSave}
                disabled={updateMutation.isPending || Object.keys(editingCosts).length === 0}
                className="flex items-center gap-2 rounded-lg bg-blue-600 px-6 py-2 font-medium text-white hover:bg-blue-700 disabled:bg-slate-700 disabled:text-slate-500 transition"
              >
                {updateMutation.isPending ? (
                  <>
                    <Loader2 size={18} className="animate-spin" />
                    Saving...
                  </>
                ) : (
                  <>
                    <Save size={18} />
                    Save Changes
                  </>
                )}
              </button>
              <button
                onClick={() => {
                  setEditingCosts({})
                  setReason('')
                }}
                className="rounded-lg border border-slate-600 px-6 py-2 font-medium text-slate-300 hover:bg-slate-800 transition"
              >
                Cancel
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
