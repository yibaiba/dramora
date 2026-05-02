import { useState } from 'react'
import { X, Loader2 } from 'lucide-react'
import { useChargeWallet } from '../../api/hooks'
import type { ChargeWalletRequest } from '../../api/types'

interface ChargeWalletDialogProps {
  isOpen: boolean
  onClose: () => void
  onSuccess?: () => void
}

export default function ChargeWalletDialog({ isOpen, onClose, onSuccess }: ChargeWalletDialogProps) {
  const [amount, setAmount] = useState<string>('100')
  const [description, setDescription] = useState<string>('User Charge')
  const chargeWallet = useChargeWallet()
  const isLoading = chargeWallet.isPending

  const handleCharge = async () => {
    const chargeAmount = parseInt(amount, 10)
    if (!chargeAmount || chargeAmount <= 0) {
      alert('Please enter a valid amount')
      return
    }

    const request: ChargeWalletRequest = {
      amount: chargeAmount,
      description: description.trim() || 'User Charge',
      payment_method: 'mock', // MVP: 模拟支付
    }

    await chargeWallet.mutateAsync(request, {
      onSuccess: () => {
        alert(`Successfully charged ${chargeAmount} credits!`)
        setAmount('100')
        setDescription('User Charge')
        onClose()
        onSuccess?.()
      },
      onError: (error: any) => {
        alert(`Charge failed: ${error.message || 'Unknown error'}`)
      },
    })
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-full max-w-md rounded-lg bg-white dark:bg-slate-900 shadow-xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-700 p-6">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Charge Credits</h2>
          <button
            onClick={onClose}
            className="text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-4">
          {/* Amount Input */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              Amount (Credits)
            </label>
            <input
              type="number"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
              min="1"
              disabled={isLoading}
            />
            <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">Minimum 1 credit</p>
          </div>

          {/* Description Input */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              Description (Optional)
            </label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="e.g., Monthly subscription"
              disabled={isLoading}
            />
          </div>

          {/* Note */}
          <div className="p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
            <p className="text-xs text-blue-600 dark:text-blue-400">
              💡 MVP: This is a mock charge. In production, you'll be directed to a payment gateway.
            </p>
          </div>
        </div>

        {/* Footer */}
        <div className="flex gap-3 border-t border-slate-200 dark:border-slate-700 p-6">
          <button
            onClick={onClose}
            disabled={isLoading}
            className="flex-1 py-2 px-4 border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-300 font-medium rounded-lg hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={handleCharge}
            disabled={isLoading}
            className="flex-1 py-2 px-4 bg-orange-500 hover:bg-orange-600 text-white font-medium rounded-lg transition-colors flex items-center justify-center gap-2 disabled:opacity-50"
          >
            {isLoading && <Loader2 className="w-4 h-4 animate-spin" />}
            {isLoading ? 'Processing...' : 'Charge'}
          </button>
        </div>
      </div>
    </div>
  )
}
