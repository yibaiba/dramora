import { Loader2 } from 'lucide-react'
import { useWallet, useOperationCosts } from '../../api/hooks'
import WalletBalanceCard from '../components/WalletBalanceCard'
import OperationCostsTable from '../components/OperationCostsTable'

export default function WalletPage() {
  const { data: wallet, isLoading: walletLoading, refetch: refetchWallet } = useWallet()
  const { data: costs, isLoading: costsLoading } = useOperationCosts()

  if (walletLoading || costsLoading) {
    return (
      <div className="flex items-center justify-center w-full h-screen">
        <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
      </div>
    )
  }

  return (
    <div className="w-full max-w-6xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-slate-900 dark:text-white mb-2">Wallet</h1>
        <p className="text-slate-600 dark:text-slate-400">Manage your credits and view operation costs</p>
      </div>

      {/* Balance Card */}
      <div className="mb-12">
        <WalletBalanceCard wallet={wallet} onChargeSuccess={() => refetchWallet()} />
      </div>

      {/* Operation Costs Table */}
      <div>
        <h2 className="text-xl font-semibold text-slate-900 dark:text-white mb-4">Operation Costs</h2>
        <OperationCostsTable costs={costs} />
      </div>
    </div>
  )
}
