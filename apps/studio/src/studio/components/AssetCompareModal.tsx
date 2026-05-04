import { X, ChevronLeft, ChevronRight } from 'lucide-react'
import type { Asset } from '../../api/types'

interface AssetCompareModalProps {
  isOpen: boolean
  assets: Asset[] // All available assets (for navigation)
  assetIds: [string, string] // The pair being compared
  onClose: () => void
  onNavigate: (direction: 'prev' | 'next') => void
}

export function AssetCompareModal({
  isOpen,
  assets,
  assetIds: [asset1Id, asset2Id],
  onClose,
  onNavigate,
}: AssetCompareModalProps) {
  if (!isOpen) return null

  const asset1 = assets.find((a) => a.id === asset1Id)
  const asset2 = assets.find((a) => a.id === asset2Id)

  if (!asset1 || !asset2) return null

  return (
    <>
      <div
        className="edit-modal-overlay"
        onClick={onClose}
        style={{ zIndex: 1000 }}
        aria-label="关闭对比"
        role="button"
        tabIndex={0}
        onKeyDown={(e) => {
          if (e.key === 'Escape') onClose()
        }}
      >
        <div
          className="compare-modal"
          onClick={(e) => e.stopPropagation()}
          style={{
            position: 'fixed',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            width: '90vw',
            maxWidth: '1200px',
            height: '85vh',
            backgroundColor: 'rgb(19, 22, 33)',
            borderRadius: '12px',
            border: '1px solid rgba(148, 163, 184, 0.2)',
            display: 'flex',
            flexDirection: 'column',
            boxShadow: '0 20px 60px rgba(0, 0, 0, 0.5)',
            zIndex: 1001,
          }}
        >
          {/* Header */}
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              padding: '20px 24px',
              borderBottom: '1px solid rgba(148, 163, 184, 0.1)',
            }}
          >
            <h2 style={{ margin: 0, fontSize: '18px', fontWeight: 600 }}>资产对比</h2>
            <button
              onClick={onClose}
              className="ghost-button"
              title="关闭"
              style={{
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                color: 'rgba(148, 163, 184, 0.7)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <X size={20} />
            </button>
          </div>

          {/* Content */}
          <div
            style={{
              display: 'flex',
              flex: 1,
              overflow: 'hidden',
            }}
          >
            {/* Left asset */}
            <div
              style={{
                flex: 1,
                display: 'flex',
                flexDirection: 'column',
                padding: '24px',
                borderRight: '1px solid rgba(148, 163, 184, 0.1)',
                overflowY: 'auto',
              }}
            >
              <AssetCard asset={asset1} />
            </div>

            {/* Right asset */}
            <div
              style={{
                flex: 1,
                display: 'flex',
                flexDirection: 'column',
                padding: '24px',
                overflowY: 'auto',
              }}
            >
              <AssetCard asset={asset2} />
            </div>
          </div>

          {/* Footer with navigation */}
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              padding: '16px 24px',
              borderTop: '1px solid rgba(148, 163, 184, 0.1)',
              backgroundColor: 'rgba(19, 22, 33, 0.5)',
            }}
          >
            <button
              onClick={() => onNavigate('prev')}
              className="ghost-button"
              title="上一对"
              style={{
                background: 'rgba(148, 163, 184, 0.1)',
                border: 'none',
                borderRadius: '6px',
                padding: '8px 12px',
                cursor: 'pointer',
                color: 'rgba(148, 163, 184, 0.9)',
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
                fontSize: '14px',
              }}
            >
              <ChevronLeft size={16} />
              上一对
            </button>
            <span style={{ fontSize: '12px', color: 'rgba(148, 163, 184, 0.7)' }}>
              在所有资产中导航
            </span>
            <button
              onClick={() => onNavigate('next')}
              className="ghost-button"
              title="下一对"
              style={{
                background: 'rgba(148, 163, 184, 0.1)',
                border: 'none',
                borderRadius: '6px',
                padding: '8px 12px',
                cursor: 'pointer',
                color: 'rgba(148, 163, 184, 0.9)',
                display: 'flex',
                alignItems: 'center',
                gap: '6px',
                fontSize: '14px',
              }}
            >
              下一对
              <ChevronRight size={16} />
            </button>
          </div>
        </div>
      </div>
    </>
  )
}

function AssetCard({ asset }: { asset: Asset }) {
  const kindLabels = {
    character: '角色',
    scene: '场景',
    prop: '道具',
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
      {/* Thumbnail */}
      <div
        style={{
          width: '100%',
          paddingBottom: '100%',
          position: 'relative',
          backgroundColor: 'rgba(148, 163, 184, 0.1)',
          borderRadius: '8px',
          overflow: 'hidden',
          border: '1px solid rgba(148, 163, 184, 0.2)',
        }}
      >
        <div
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            width: '100%',
            height: '100%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: 'rgba(148, 163, 184, 0.5)',
          }}
        >
          无缩略图
        </div>
      </div>

      {/* Details */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
        {/* Name */}
        <div>
          <span style={{ fontSize: '12px', color: 'rgba(148, 163, 184, 0.6)' }}>名称</span>
          <p style={{ margin: '4px 0 0 0', fontSize: '16px', fontWeight: 600 }}>
            {asset.purpose || `Asset ${asset.id.slice(0, 8)}`}
          </p>
        </div>

        {/* Type */}
        <div>
          <span style={{ fontSize: '12px', color: 'rgba(148, 163, 184, 0.6)' }}>类型</span>
          <p style={{ margin: '4px 0 0 0', fontSize: '14px' }}>
            {kindLabels[asset.kind as keyof typeof kindLabels] || asset.kind}
          </p>
        </div>

        {/* Status */}
        <div>
          <span style={{ fontSize: '12px', color: 'rgba(148, 163, 184, 0.6)' }}>状态</span>
          <p style={{ margin: '4px 0 0 0', fontSize: '12px', color: 'rgba(148, 163, 184, 0.7)' }}>
            {asset.status}
          </p>
        </div>

        {/* Created At */}
        <div>
          <span style={{ fontSize: '12px', color: 'rgba(148, 163, 184, 0.6)' }}>创建时间</span>
          <p style={{ margin: '4px 0 0 0', fontSize: '12px', color: 'rgba(148, 163, 184, 0.7)' }}>
            {new Date(asset.created_at).toLocaleString('zh-CN')}
          </p>
        </div>

        {/* URI (if available) */}
        {asset.uri && (
          <div>
            <span style={{ fontSize: '12px', color: 'rgba(148, 163, 184, 0.6)' }}>资源类型</span>
            <p style={{ margin: '4px 0 0 0', fontSize: '12px', color: 'rgba(148, 163, 184, 0.7)' }}>
              {asset.kind}
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
