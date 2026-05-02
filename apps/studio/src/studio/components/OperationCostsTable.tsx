import {
  Zap,
  MessageSquare,
  BookOpen,
  Image,
  Video,
  Wand2,
  BarChart3,
} from 'lucide-react'
import type { OperationCost, OperationType } from '../../api/types'

interface OperationCostsTableProps {
  costs?: OperationCost[]
}

const operationIcons: Record<string, React.ReactNode> = {
  chat: <MessageSquare className="w-5 h-5" />,
  story_analysis: <BookOpen className="w-5 h-5" />,
  image_generation: <Image className="w-5 h-5" />,
  video_generation: <Video className="w-5 h-5" />,
  storyboard_edit: <Wand2 className="w-5 h-5" />,
  character_edit: <BarChart3 className="w-5 h-5" />,
  scene_edit: <Zap className="w-5 h-5" />,
}

const operationLabels: Record<string, string> = {
  chat: 'Chat',
  story_analysis: 'Story Analysis',
  image_generation: 'Image Generation',
  video_generation: 'Video Generation',
  storyboard_edit: 'Storyboard Edit',
  character_edit: 'Character Edit',
  scene_edit: 'Scene Edit',
}

export default function OperationCostsTable({ costs }: OperationCostsTableProps) {
  if (!costs || costs.length === 0) {
    return (
      <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-8">
        <p className="text-center text-slate-500 dark:text-slate-400">No operation costs available</p>
      </div>
    )
  }

  // Sort by cost descending
  const sortedCosts = [...costs].sort((a, b) => b.cost - a.cost)

  return (
    <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50">
              <th className="px-6 py-4 text-left text-sm font-semibold text-slate-900 dark:text-white">
                Operation
              </th>
              <th className="px-6 py-4 text-left text-sm font-semibold text-slate-900 dark:text-white">
                Description
              </th>
              <th className="px-6 py-4 text-right text-sm font-semibold text-slate-900 dark:text-white">
                Cost
              </th>
            </tr>
          </thead>
          <tbody>
            {sortedCosts.map((item, idx) => (
              <tr
                key={item.type}
                className={`border-b transition-colors duration-200 ${
                  idx % 2 === 0
                    ? 'bg-white dark:bg-slate-900'
                    : 'bg-slate-50 dark:bg-slate-800/30'
                } hover:bg-slate-100 dark:hover:bg-slate-800`}
              >
                <td className="px-6 py-4">
                  <div className="flex items-center gap-3">
                    <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded">
                      <div className="text-blue-600 dark:text-blue-400">
                        {operationIcons[item.type] || <Zap className="w-5 h-5" />}
                      </div>
                    </div>
                    <span className="font-medium text-slate-900 dark:text-white">
                      {operationLabels[item.type] || item.type}
                    </span>
                  </div>
                </td>
                <td className="px-6 py-4">
                  <p className="text-sm text-slate-600 dark:text-slate-400">
                    {getOperationDescription(item.type as OperationType)}
                  </p>
                </td>
                <td className="px-6 py-4 text-right">
                  <div className="flex items-center justify-end gap-2">
                    <span className="font-semibold text-orange-600 dark:text-orange-400 text-lg">
                      {item.cost}
                    </span>
                    <span className="text-sm text-slate-500 dark:text-slate-400">积分</span>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Footer Info */}
      <div className="px-6 py-4 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/30">
        <p className="text-xs text-slate-600 dark:text-slate-400">
          Credits are automatically deducted upon successful completion of operations. Operations that fail are not
          charged.
        </p>
      </div>
    </div>
  )
}

function getOperationDescription(type: OperationType): string {
  const descriptions: Record<OperationType, string> = {
    chat: 'AI-powered conversation and content generation',
    story_analysis: 'Analyze and generate story outlines and structure',
    image_generation: 'Generate visual content and illustrations',
    video_generation: 'Create and process video content',
    storyboard_edit: 'Edit and refine storyboard sequences',
    character_edit: 'Modify character properties and details',
    scene_edit: 'Adjust scene settings and composition',
  }
  return descriptions[type] || 'Production operation'
}
