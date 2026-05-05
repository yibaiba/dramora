import { AlertCircle, Loader2 } from 'lucide-react'
import { useWorkspaces } from '../../api/hooks'
import type { Workspace } from '../../api/types'

export function WorkspacePage() {
  const { data: workspaces = [], isLoading, error } = useWorkspaces()

  return (
    <section className="studio-page workspace-page" aria-labelledby="workspace-title">
      <div className="board-header">
        <div>
          <h1 id="workspace-title">工作空间</h1>
          <p className="text-sm text-gray-400">浏览和管理团队工作空间</p>
        </div>
      </div>

      {error && (
        <div className="bg-red-900/20 border border-red-700 rounded-lg p-4 mb-4 flex items-start gap-3">
          <AlertCircle className="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" aria-hidden="true" />
          <div>
            <p className="text-sm font-medium text-red-300">加载失败</p>
            <p className="text-sm text-red-200">{error.message}</p>
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center h-64">
          <div className="flex flex-col items-center gap-3">
            <Loader2 className="w-8 h-8 text-gray-400 animate-spin" aria-hidden="true" />
            <p className="text-sm text-gray-400">加载工作空间中...</p>
          </div>
        </div>
      ) : workspaces.length === 0 ? (
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <p className="text-lg font-medium text-gray-300">暂无工作空间</p>
            <p className="text-sm text-gray-500 mt-2">你的组织中还没有工作空间</p>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {workspaces.map((workspace) => (
            <WorkspaceCard key={workspace.id} workspace={workspace} />
          ))}
        </div>
      )}
    </section>
  )
}

function WorkspaceCard({ workspace }: { workspace: Workspace }) {
  const createdDate = new Date(workspace.created_at)
  const formattedDate = createdDate.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })

  return (
    <div
      className="bg-gray-900/50 border border-gray-800 rounded-lg p-4 hover:bg-gray-900/70 transition-colors cursor-pointer group"
      role="article"
      aria-label={`${workspace.name} 工作空间`}
    >
      <div className="flex-1">
        <h3 className="text-lg font-semibold text-gray-100 group-hover:text-white transition-colors">
          {workspace.name}
        </h3>
        {workspace.description && (
          <p className="text-sm text-gray-400 mt-2 line-clamp-2">{workspace.description}</p>
        )}
      </div>

      <div className="flex items-center gap-4 mt-4 pt-4 border-t border-gray-800">
        <div className="flex-1">
          <div className="text-xs text-gray-500">成员</div>
          <div className="text-sm font-medium text-gray-300">{workspace.members_count}</div>
        </div>
        <div className="flex-1">
          <div className="text-xs text-gray-500">项目</div>
          <div className="text-sm font-medium text-gray-300">{workspace.projects_count}</div>
        </div>
        <div className="flex-1">
          <div className="text-xs text-gray-500">创建于</div>
          <div className="text-sm font-medium text-gray-300">{formattedDate}</div>
        </div>
      </div>
    </div>
  )
}
