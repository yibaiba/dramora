import { Trash2, Download, X } from 'lucide-react'
import { useState } from 'react'
import { useDeleteAsset } from '../../api/hooks'
import type { Asset } from '../../api/types'
import { useEpisodeAssets } from '../../api/hooks'
import { ConfirmDeleteDialog } from './ConfirmDeleteDialog'
import { ExportProgressDialog } from './ExportProgressDialog'
import { DeleteProgressDialog } from './DeleteProgressDialog'

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
  const { data: assets = [] } = useEpisodeAssets(episodeId)
  const deleteAssetMutation = useDeleteAsset()

  const selectedAssets = assets.filter((a: Asset) => selectedAssetIds.has(a.id))

  const handleConfirmDelete = async () => {
    setShowDeleteConfirm(false)
    setShowDeleteProgress(true)
  }

  const handleStartExport = () => {
    setShowExportProgress(true)
  }

  return (
    <>
      <div className="selection-toolbar">
        <div className="toolbar-info">
          <span className="selection-count">已选 {selectedCount} 个资产</span>
        </div>
        <div className="toolbar-actions">
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
    </>
  )
}
