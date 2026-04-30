import {
  BookOpenText,
  Boxes,
  Download,
  Home,
  Layers3,
  Settings,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

export const studioRoutePaths = {
  adminSettings: '/admin/settings',
  assetsGraph: '/assets-graph',
  home: '/',
  storyAnalysis: '/story-analysis',
  storyboard: '/storyboard',
  timelineExport: '/timeline-export',
} as const

export type StudioNavItem = {
  description: string
  disabled?: boolean
  icon: LucideIcon
  key: keyof typeof studioRoutePaths
  label: string
  path: string
}

export const studioNavItems: StudioNavItem[] = [
  {
    description: '项目总览、生产信号和页面跳转入口。',
    icon: Home,
    key: 'home',
    label: 'Home',
    path: studioRoutePaths.home,
  },
  {
    description: '录入故事源、启动多 Agent 解析并查看结果。',
    icon: BookOpenText,
    key: 'storyAnalysis',
    label: 'Story Analysis',
    path: studioRoutePaths.storyAnalysis,
  },
  {
    description: '管理分镜卡、提示词和镜头级审批动作。',
    icon: Boxes,
    key: 'storyboard',
    label: 'Storyboard',
    path: studioRoutePaths.storyboard,
  },
  {
    description: '浏览故事图谱、候选资产并锁定参考素材。',
    icon: Layers3,
    key: 'assetsGraph',
    label: 'Assets / Graph',
    path: studioRoutePaths.assetsGraph,
  },
  {
    description: '编排时间线、保存剪辑并发起导出。',
    icon: Download,
    key: 'timelineExport',
    label: 'Timeline / Export',
    path: studioRoutePaths.timelineExport,
  },
  {
    description: '管理 AI 端点配置和积分设置。',
    icon: Settings,
    key: 'adminSettings',
    label: 'Settings',
    path: studioRoutePaths.adminSettings,
  },
]
