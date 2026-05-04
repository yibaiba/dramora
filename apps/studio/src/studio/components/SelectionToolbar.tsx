import { Trash2, Download, X, GitCompare } from 'lucide-react'
import { useState, useMemo } from 'react'
import { useDeleteAsset } from '../../api/hooks'
import type { Asset } from '../../api/types'
import { useEpisodeAssets } from '../../api/hooks'
import { ConfirmDeleteDialog } from './ConfirmDeleteDialog'
import { ExportProgressDialog } from './ExportProgressDialog'
import { DeleteProgressDialog } from './DeleteProgressDialog'
import { AssetCompareModal } from './AssetCompareModal'

export function SelectionToolbar({
  selectedCount,
  onClearSelection,
  selectedAssetIds,
  episodeId,
  onDeleteSuccess,
}: {
  selectedCount: number
  onClearSelection: () => void
  selectedAssetIds: Set<string>
  episodeId: string
  onDeleteSuccess: () => void
}) {
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [showDeleteProgress, setShowDeleteProgress] = useState(false)
  const [showExportProgress, setShowExportProgress] = useState(false)
  const [showCompare, setShowCompare] = useState(false)
  const [compareIndex, setCompareIndex] = useState(0)
  const { data: assets = [] } = useEpisodeAssets(episodeId)
  const deleteAssetMutation = useDeleteAsset()

  // 使用 useMemo 缓存 selectedAssets，避免频繁重新计算
  const selectedAssets = useMemo(
    () => assets.filter((a: Asset) => selectedAssetIds.has(a.id)),
    [assets, selectedAssetIds],
  )

  // Generate all possible pairs from selected assets for navigation
  const assetPairs = useMemo(() => {
    const ids = Array.from(selectedAssetIds)
    const pairs: [string, string][] = []
    for (let i = 0; i < ids.length - 1; i++) {
      for (let j = i + 1; j < ids.length; j++) {
        pairs.push([ids[i], ids[j]])
      }
    }
    return pairs
  }, [selectedAssetIds])

  const currentPair = assetPairs[compareIndex] || (assetPairs.length > 0 ? assetPairs[0] : null)

  const handleConfirmDelete = async () => {
    setShowDeleteConfirm(false)
    setShowDeleteProgress(true)
  }

  const handleStartExport = () => {
    setShowExportProgress(true)
  }

  const handleCompare = () => {
    setCompareIndex(0)
    setShowCompare(true)
  }

  const handleNavigateComparison = (direction: 'prev' | 'next') => {
    if (direction === 'prev') {
      setCompareIndex((prev) => (prev === 0 ? assetPairs.length - 1 : prev - 1))
    } else {
      setCompareIndex((prev) => (prev === assetPairs.length - 1 ? 0 : prev + 1))
    }
  }

  return (
    <>
      <div className="selection-toolbar">
        <div className="toolbar-info">
          <span className="selection-count">已选 {selectedCount} 个资产</span>
        </div>
        <div className="toolbar-actions">
          <button
            onClick={handleCompare}
            className="toolbar-button toolbar-button--primary"
            type="button"
            title={selectedCount === 2 ? '对比这两个资产' : selectedCount < 2 ? '需要至少选中 2 个资产' : '只能对比 2 个资产，请调整选择'}
            disabled={selectedCount < 2}
          >
            <GitCompare size={16} />
            对比
          </button>
          <button
            onClick={() => setShowDeleteConfirm(true)}
            className="toolbar-button toolbar-button--danger"
            type="button"
            title="删除选中的资产"
            disabled={deleteAssetMutation.isPending}
          >
            <Trash2 size={16} />
            删除
          </button>
          <button
            onClick={handleStartExport}
            className="toolbar-button toolbar-button--primary"
            type="button"
            title="导出选中的资产为 ZIP"
          >
            <Download size={16} />
            导出
          </button>
          <button
            onClick={onClearSelection}
            className="toolbar-button toolbar-button--secondary"
            type="button"
            title="清空选择"
          >
            <X size={16} />
            清空
          </button>
        </div>
      </div>

      {showDeleteConfirm && (
        <ConfirmDeleteDialog
          selectedCount={selectedCount}
          onConfirm={handleConfirmDelete}
          onCancel={() => setShowDeleteConfirm(false)}
          isLoading={false}
        />
      )}

      {showDeleteProgress && (
        <DeleteProgressDialog
          selectedAssets={selectedAssets}
          episodeId={episodeId}
          onClose={() => {
            setShowDeleteProgress(false)
            onDeleteSuccess()
          }}
        />
      )}

      {showExportProgress && (
        <ExportProgressDialog
          selectedAssets={selectedAssets}
          episodeId={episodeId}
          onClose={() => setShowExportProgress(false)}
        />
      )}

      {showCompare && currentPair && (
        <AssetCompareModal
          isOpen={showCompare}
          assets={assets}
          assetIds={currentPair}
          onClose={() => setShowCompare(false)}
          onNavigate={handleNavigateComparison}
        />
      )}
    </>
  )
}
