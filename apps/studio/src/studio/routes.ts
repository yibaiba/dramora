import {
  Activity,
  Bell,
  BookOpenText,
  Boxes,
  Coins,
  Download,
  Home,
  History,
  Image,
  KeyRound,
  Layers3,
  Mail,
  Settings,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

export const studioRoutePaths = {
  adminSettings: '/admin/settings',
  assetsGraph: '/assets-graph',
  gallery: '/gallery',
  home: '/',
  mySessions: '/account/sessions',
  notifications: '/notifications',
  organizationInvitations: '/admin/invitations',
  queue: '/queue',
  storyAnalysis: '/story-analysis',
  storyboard: '/storyboard',
  timelineExport: '/timeline-export',
  wallet: '/wallet',
  transactions: '/transactions',
  workerMetrics: '/admin/worker-metrics',
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
    description: '网格视图浏览所有生成的素材，支持类型和状态筛选。',
    icon: Image,
    key: 'gallery',
    label: 'Gallery',
    path: studioRoutePaths.gallery,
  },
  {
    description: '实时监控所有生成任务的进度和状态。',
    icon: Activity,
    key: 'queue',
    label: 'Queue',
    path: studioRoutePaths.queue,
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
  {
    description: '为协作者签发组织邀请（owner / admin）。',
    icon: Mail,
    key: 'organizationInvitations',
    label: 'Invitations',
    path: studioRoutePaths.organizationInvitations,
  },
  {
    description: '查看并吊销当前账号的登录会话。',
    icon: KeyRound,
    key: 'mySessions',
    label: 'Sessions',
    path: studioRoutePaths.mySessions,
  },
  {
    description: '查看积分余额、充值/扣费流水（owner / admin 可调整）。',
    icon: Coins,
    key: 'wallet',
    label: 'Wallet',
    path: studioRoutePaths.wallet,
  },
  {
    description: '查看完整的交易历史和流水明细。',
    icon: History,
    key: 'transactions',
    label: 'Transactions',
    path: studioRoutePaths.transactions,
  },
  {
    description: '查看系统通知：钱包操作、邀请、配置变更。',
    icon: Bell,
    key: 'notifications',
    label: 'Notifications',
    path: studioRoutePaths.notifications,
  },
  {
    description: '查看 worker 组织上下文跳过指标（owner / admin）。',
    icon: Activity,
    key: 'workerMetrics',
    label: 'Worker Metrics',
    path: studioRoutePaths.workerMetrics,
  },
]
