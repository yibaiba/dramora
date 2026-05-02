import { Library, X } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useEpisodeAssets } from '../../api/hooks'
import type { Asset, AssetStatus } from '../../api/types'
import { StatePlaceholder } from '../components/StatePlaceholder'
import { useStudioSelection } from '../hooks/useStudioSelection'
import { studioRoutePaths } from '../routes'

type AssetKind = 'character' | 'scene' | 'prop'
type FilterKind = 'all' | AssetKind
type FilterStatus = 'all' | 'generating' | 'ready' | 'failed'

export function GalleryPage() {
  const { activeEpisode } = useStudioSelection()
  const { data: assets = [], isLoading } = useEpisodeAssets(activeEpisode?.id)
  const [filterKind, setFilterKind] = useState<FilterKind>('all')
  const [filterStatus, setFilterStatus] = useState<FilterStatus>('all')

  const filtered = useMemo(() => {
    return assets.filter((asset) => {
      if (filterKind !== 'all' && asset.kind !== filterKind) return false
      if (filterStatus !== 'all' && asset.status !== filterStatus) return false
      return true
    })
  }, [assets, filterKind, filterStatus])

  const assetsByStatus = {
    all: assets.filter((a) => a.status === 'generating' || a.status === 'ready' || a.status === 'failed').length,
    generating: assets.filter((a) => a.status === 'generating').length,
    ready: assets.filter((a) => a.status === 'ready').length,
    failed: assets.filter((a) => a.status === 'failed').length,
  }

  const stats = useMemo(
    () => ({
      total: assets.length,
      byKind: {
        character: assets.filter((a) => a.kind === 'character').length,
        scene: assets.filter((a) => a.kind === 'scene').length,
        prop: assets.filter((a) => a.kind === 'prop').length,
      },
      byStatus: assetsByStatus,
    }),
    [assets],
  )

  const clearFilters = () => {
    setFilterKind('all')
    setFilterStatus('all')
  }

  const activeFilterCount = (filterKind !== 'all' ? 1 : 0) + (filterStatus !== 'all' ? 1 : 0)

  return (
    <section className="studio-page gallery-page" aria-labelledby="gallery-title">
      <div className="board-header">
        <div>
          <h1 id="gallery-title">素材库</h1>
          <span>浏览、筛选和管理所有已生成的素材资产。</span>
        </div>
        <div className="board-actions">
          <Link className="hero-secondary-action" to={studioRoutePaths.assetsGraph}>
            <span aria-hidden="true">←</span>
            回到 Assets / Graph
          </Link>
        </div>
      </div>

      <div className="dashboard-grid">
        <article className="surface-card">
          <span className="section-kicker">Total assets</span>
          <strong>{stats.total} 个素材</strong>
          <p>当前 episode 已生成的全部资产文件。</p>
        </article>
        <article className="surface-card">
          <span className="section-kicker">By type</span>
          <strong>
            {stats.byKind.character} 角 · {stats.byKind.scene} 景 · {stats.byKind.prop} 道
          </strong>
          <p>角色、场景、道具资产分布。</p>
        </article>
        <article className="surface-card">
          <span className="section-kicker">Ready</span>
          <strong>{stats.byStatus.ready} 已锁定</strong>
          <p>可用于后续视频生成的素材。</p>
        </article>
      </div>

      <article className="surface-card">
        <div className="panel-title-row">
          <div>
            <span>Gallery</span>
            <strong>素材网格视图 · {filtered.length} 个结果</strong>
          </div>
          {activeFilterCount > 0 && (
            <button
              className="gallery-clear-filters"
              onClick={clearFilters}
              type="button"
              title="清空所有筛选条件"
            >
              <X size={16} aria-hidden="true" />
              清空筛选
            </button>
          )}
        </div>

        <div className="gallery-filters">
          <div className="filter-group">
            <span className="filter-label">类型</span>
            <div className="filter-buttons">
              <button
                className={filterKind === 'all' ? 'filter-btn active' : 'filter-btn'}
                onClick={() => setFilterKind('all')}
                type="button"
              >
                全部 ({stats.byKind.character + stats.byKind.scene + stats.byKind.prop})
              </button>
              <button
                className={filterKind === 'character' ? 'filter-btn active' : 'filter-btn'}
                onClick={() => setFilterKind('character')}
                type="button"
              >
                角色 ({stats.byKind.character})
              </button>
              <button
                className={filterKind === 'scene' ? 'filter-btn active' : 'filter-btn'}
                onClick={() => setFilterKind('scene')}
                type="button"
              >
                场景 ({stats.byKind.scene})
              </button>
              <button
                className={filterKind === 'prop' ? 'filter-btn active' : 'filter-btn'}
                onClick={() => setFilterKind('prop')}
                type="button"
              >
                道具 ({stats.byKind.prop})
              </button>
            </div>
          </div>

          <div className="filter-group">
            <span className="filter-label">状态</span>
            <div className="filter-buttons">
              <button
                className={filterStatus === 'all' ? 'filter-btn active' : 'filter-btn'}
                onClick={() => setFilterStatus('all')}
                type="button"
              >
                全部 ({stats.byStatus.all})
              </button>
              <button
                className={filterStatus === 'ready' ? 'filter-btn active' : 'filter-btn'}
                onClick={() => setFilterStatus('ready')}
                type="button"
              >
                已锁定 ({stats.byStatus.ready})
              </button>
              <button
                className={filterStatus === 'generating' ? 'filter-btn active' : 'filter-btn'}
                onClick={() => setFilterStatus('generating')}
                type="button"
              >
                生成中 ({stats.byStatus.generating})
              </button>
              <button
                className={filterStatus === 'failed' ? 'filter-btn active' : 'filter-btn'}
                onClick={() => setFilterStatus('failed')}
                type="button"
              >
                失败 ({stats.byStatus.failed})
              </button>
            </div>
          </div>
        </div>

        {isLoading ? (
          <StatePlaceholder
            tone="loading"
            title="加载中..."
            description="正在获取素材列表。"
          />
        ) : assets.length === 0 ? (
          <StatePlaceholder
            tone="empty"
            icon={Library}
            title="还没有素材"
            description="生成资产图谱并创建候选资产后，这里会显示素材库。"
          />
        ) : filtered.length === 0 ? (
          <StatePlaceholder
            tone="empty"
            title="当前筛选条件下没有素材"
            description="请调整筛选条件。"
          />
        ) : (
          <div className="gallery-grid">
            {filtered.map((asset) => (
              <GalleryAssetCard key={asset.id} asset={asset} />
            ))}
          </div>
        )}
      </article>
    </section>
  )
}

function GalleryAssetCard({ asset }: { asset: Asset }) {
  const statusLabel: Record<AssetStatus, string> = {
    draft: '草稿',
    generating: '生成中',
    ready: '已锁定',
    failed: '失败',
    archived: '已归档',
  }

  const kindLabel: Record<string, string> = {
    character: '角色',
    scene: '场景',
    prop: '道具',
  }

  const statusTone = asset.status === 'ready' ? 'success' : asset.status === 'failed' ? 'error' : 'neutral'

  return (
    <article className={`gallery-asset-card tone-${statusTone}`}>
      <div className="asset-header">
        <div>
          <span className="asset-kind-badge">{kindLabel[asset.kind] ?? asset.kind}</span>
          <h3>{asset.purpose}</h3>
        </div>
        <span className={`asset-status-badge status-${asset.status}`}>{statusLabel[asset.status]}</span>
      </div>

      <p className="asset-description">
        {asset.kind} 素材 · {asset.purpose}
      </p>

      <div className="asset-meta">
        <span className="asset-uri" title={asset.uri}>
          {asset.uri.replace('manmu://', '')}
        </span>
      </div>
    </article>
  )
}
