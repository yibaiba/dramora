import { useState } from 'react'
import { Coins, TrendingUp } from 'lucide-react'
import type { WalletSnapshot } from '../../api/types'
import ChargeWalletDialog from './ChargeWalletDialog'

interface WalletBalanceCardProps {
  wallet?: WalletSnapshot
}

export default function WalletBalanceCard({ wallet }: WalletBalanceCardProps) {
  const [isChargeDialogOpen, setIsChargeDialogOpen] = useState(false)
  const balance = wallet?.wallet?.balance ?? 0
  const lastUpdated = wallet?.wallet?.updated_at ? new Date(wallet.wallet.updated_at).toLocaleString() : 'Never'

  return (
    <>
      <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 overflow-hidden shadow-sm hover:shadow-md transition-shadow duration-200">
        <div className="p-6">
          {/* Header */}
          <div className="flex items-start justify-between mb-6">
            <div>
              <p className="text-sm font-medium text-slate-600 dark:text-slate-400 mb-1">Available Credits</p>
              <div className="flex items-baseline gap-2">
                <span className="text-4xl font-bold text-slate-900 dark:text-white">{balance}</span>
                <span className="text-lg text-slate-500 dark:text-slate-400">积分</span>
              </div>
            </div>
            <div className="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
              <Coins className="w-6 h-6 text-blue-600 dark:text-blue-400" />
            </div>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-2 gap-4 mb-6">
            <div className="p-3 bg-slate-50 dark:bg-slate-800 rounded">
              <p className="text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">Status</p>
              <p className="text-sm font-semibold text-slate-900 dark:text-white">
                {balance > 0 ? 'Active' : 'Low'}
              </p>
            </div>
            <div className="p-3 bg-slate-50 dark:bg-slate-800 rounded">
              <p className="text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">Last Updated</p>
              <p className="text-xs text-slate-500 dark:text-slate-400 truncate">{lastUpdated}</p>
            </div>
          </div>

          {/* Action Button */}
          <button
            onClick={() => setIsChargeDialogOpen(true)}
            className="w-full py-3 px-4 bg-orange-500 hover:bg-orange-600 text-white font-medium rounded-lg transition-colors duration-150 flex items-center justify-center gap-2"
          >
            <TrendingUp className="w-4 h-4" />
            Charge Credits
          </button>

          {/* Help Text */}
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-3">
            Credits are automatically deducted when you use production features. Charge your account to continue
            creating.
          </p>
        </div>
      </div>

      {/* Charge Dialog */}
      <ChargeWalletDialog
        isOpen={isChargeDialogOpen}
        onClose={() => setIsChargeDialogOpen(false)}
      />
    </>
  )
}
