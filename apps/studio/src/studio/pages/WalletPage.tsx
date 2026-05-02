import { useEffect, useState } from 'react'
import { Loader2, CheckCircle, AlertCircle, X } from 'lucide-react'
import { useWallet, useOperationCosts } from '../../api/hooks'
import WalletBalanceCard from '../components/WalletBalanceCard'
import OperationCostsTable from '../components/OperationCostsTable'

export default function WalletPage() {
  const { data: wallet, isLoading: walletLoading, refetch: refetchWallet } = useWallet()
  const { data: costs, isLoading: costsLoading } = useOperationCosts()
  const [paymentStatus, setPaymentStatus] = useState<'success' | 'cancel' | null>(null)

  // Handle payment callback status from URL
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const status = params.get('status') as 'success' | 'cancel' | null
    if (status) {
      setPaymentStatus(status)
      // Refetch wallet if payment was successful
      if (status === 'success') {
        refetchWallet()
      }
      // Clear the URL parameter after 5 seconds
      const timer = setTimeout(() => {
        window.history.replaceState({}, '', '/wallet')
        setPaymentStatus(null)
      }, 5000)
      return () => clearTimeout(timer)
    }
  }, [refetchWallet])

  if (walletLoading || costsLoading) {
    return (
      <div className="flex items-center justify-center w-full h-screen">
        <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
      </div>
    )
  }

  return (
    <div className="w-full max-w-6xl mx-auto px-4 py-8">
      {/* Payment Status Alert */}
      {paymentStatus === 'success' && (
        <div className="mb-6 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg flex items-start gap-3">
          <CheckCircle className="w-5 h-5 text-green-600 dark:text-green-400 flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <h3 className="font-semibold text-green-900 dark:text-green-200">Payment Successful</h3>
            <p className="text-sm text-green-700 dark:text-green-300 mt-1">Your credits have been added to your wallet.</p>
          </div>
          <button
            onClick={() => setPaymentStatus(null)}
            className="text-green-600 dark:text-green-400 hover:text-green-700 dark:hover:text-green-300"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
      )}

      {paymentStatus === 'cancel' && (
        <div className="mb-6 p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg flex items-start gap-3">
          <AlertCircle className="w-5 h-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <h3 className="font-semibold text-amber-900 dark:text-amber-200">Payment Cancelled</h3>
            <p className="text-sm text-amber-700 dark:text-amber-300 mt-1">Your payment was cancelled. No charges have been made.</p>
          </div>
          <button
            onClick={() => setPaymentStatus(null)}
            className="text-amber-600 dark:text-amber-400 hover:text-amber-700 dark:hover:text-amber-300"
          >
            <X className="w-5 h-5" />
          </button>
        </div>
      )}

      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-slate-900 dark:text-white mb-2">Wallet</h1>
        <p className="text-slate-600 dark:text-slate-400">Manage your credits and view operation costs</p>
      </div>

      {/* Balance Card */}
      <div className="mb-12">
        <WalletBalanceCard wallet={wallet} />
      </div>

      {/* Operation Costs Table */}
      <div>
        <h2 className="text-xl font-semibold text-slate-900 dark:text-white mb-4">Operation Costs</h2>
        <OperationCostsTable costs={costs} />
      </div>
    </div>
  )
}
