export type AnalysisTemplate = {
  id: string
  name: string
  description: string
  hints: string[]
  tone: string
}

export const storyAnalysisTemplates: AnalysisTemplate[] = [
  {
    description: '强调升级线、宗门关系、战斗节奏和东方奇幻意象。',
    hints: ['主角升级线明确', '宗门冲突要具体', '镜头偏好多层次长镜头'],
    id: 'xianxia',
    name: '古风仙侠',
    tone: '史诗修真',
  },
  {
    description: '突出世界观规则、都市危机、反差角色和科技视觉符号。',
    hints: ['科技设定先讲清', '冲突要带压迫感', '偏好霓虹夜景与高速调度'],
    id: 'cyberpunk',
    name: '赛博朋克',
    tone: '高对比都市',
  },
  {
    description: '适合轻剧情、人物关系推进和现实题材的情绪表达。',
    hints: ['人物动机要落地', '对话节奏更生活化', '镜头偏好近景和情绪特写'],
    id: 'modern',
    name: '现代都市',
    tone: '情绪现实',
  },
  {
    description: '强化节奏钩子、夸张情绪和可社媒传播的分镜转场。',
    hints: ['开头 3 秒先给钩子', '角色反差更强', '多用夸张构图和转场'],
    id: 'webtoon',
    name: '韩漫分镜',
    tone: '强节奏短篇',
  },
]
