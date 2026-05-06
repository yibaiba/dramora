import { BarChart3, BookOpen, Image, MessageSquare, Sparkles, Video, Wand2, Zap } from 'lucide-react'
import { useMemo } from 'react'

import type { OperationCost, OperationType } from '../../api/types'
import { StatePlaceholder } from './StatePlaceholder'

interface OperationCostsTableProps {
  costs?: OperationCost[]
}

const operationMeta: Record<
  OperationType,
  {
    description: string
    icon: typeof Zap
    label: string
  }
> = {
  character_edit: {
    description: '修改角色设定、参考资产和一致性信息',
    icon: BarChart3,
    label: '角色编辑',
  },
  chat: {
    description: '对话式生成、提问与辅助创作',
    icon: MessageSquare,
    label: '对话',
  },
  image_generation: {
    description: '生成角色、场景、素材与图像变体',
    icon: Image,
    label: '图像生成',
  },
  scene_edit: {
    description: '修改场景设定与环境约束',
    icon: Zap,
    label: '场景编辑',
  },
  storyboard_edit: {
    description: '更新分镜、镜头设定和提示词',
    icon: Wand2,
    label: '分镜编辑',
  },
  story_analysis: {
    description: '生成故事分析、结构化大纲与工作流输入',
    icon: BookOpen,
    label: '故事分析',
  },
  video_generation: {
    description: '渲染短视频、镜头视频与导出素材',
    icon: Video,
    label: '视频生成',
  },
}

export default function OperationCostsTable({ costs }: OperationCostsTableProps) {
  const sortedCosts = useMemo(() => [...(costs ?? [])].sort((a, b) => b.cost - a.cost), [costs])

  if (sortedCosts.length === 0) {
    return <StatePlaceholder tone="empty" title="暂无定价数据" description="运营后台配置后，这里会展示每项能力的扣费规则。" />
  }

  return (
    <div className="wallet-cost-grid">
      {sortedCosts.map((item) => {
        const meta = operationMeta[item.type]
        const Icon = meta?.icon ?? Sparkles
        return (
          <article key={item.type} className="wallet-cost-card">
            <div className="wallet-cost-card-header">
              <div className="wallet-cost-card-icon" aria-hidden="true">
                <Icon size={18} />
              </div>
              <div>
                <strong>{meta?.label ?? item.type}</strong>
                <p>{meta?.description ?? '默认生产能力定价。'}</p>
              </div>
            </div>
            <div className="wallet-cost-card-footer">
              <span>成功后扣费</span>
              <strong>{item.cost.toLocaleString('zh-CN')} 积分</strong>
            </div>
          </article>
        )
      })}
    </div>
  )
}
