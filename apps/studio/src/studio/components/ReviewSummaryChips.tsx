import { Link } from 'react-router-dom'
import { studioRoutePaths } from '../routes'

type ChipSide = 'storyboard' | 'assetsGraph' | 'storyAnalysis'

type ReviewSummaryChipsProps = {
  currentSide: ChipSide
  storyboardPendingCount: number
  assetsGraphPendingCount: number
  totalReturnedCount: number
  storyAnalysisLinkState?: unknown
}

function chipClass(pending: number, isCurrent: boolean): string {
  const base = ['blackboard-chip']
  if (isCurrent) base.push('blackboard-chip-current')
  if (pending > 0) base.push('blackboard-chip-warn')
  else base.push('blackboard-chip-ready')
  return base.join(' ')
}

export function ReviewSummaryChips({
  currentSide,
  storyboardPendingCount,
  assetsGraphPendingCount,
  totalReturnedCount,
  storyAnalysisLinkState,
}: ReviewSummaryChipsProps) {
  const storyboardCurrent = currentSide === 'storyboard'
  const assetsGraphCurrent = currentSide === 'assetsGraph'
  const allCleared = storyboardPendingCount === 0 && assetsGraphPendingCount === 0
  const canCloseRound = allCleared && totalReturnedCount > 0

  const storyboardLabel = `${storyboardCurrent ? '本页 · ' : ''}Storyboard 待跟进 ${storyboardPendingCount}`
  const assetsGraphLabel = `${assetsGraphCurrent ? '本页 · ' : ''}Assets / Graph 待跟进 ${assetsGraphPendingCount}`

  return (
    <div className="blackboard-chip-row">
      {!storyboardCurrent && storyboardPendingCount > 0 ? (
        <Link
          className={`${chipClass(storyboardPendingCount, false)} blackboard-chip-link`}
          to={studioRoutePaths.storyboard}
          title="去 Storyboard 处理待跟进"
        >
          {storyboardLabel}
        </Link>
      ) : (
        <span className={chipClass(storyboardPendingCount, storyboardCurrent)}>{storyboardLabel}</span>
      )}
      {!assetsGraphCurrent && assetsGraphPendingCount > 0 ? (
        <Link
          className={`${chipClass(assetsGraphPendingCount, false)} blackboard-chip-link`}
          to={studioRoutePaths.assetsGraph}
          title="去 Assets / Graph 处理待跟进"
        >
          {assetsGraphLabel}
        </Link>
      ) : (
        <span className={chipClass(assetsGraphPendingCount, assetsGraphCurrent)}>{assetsGraphLabel}</span>
      )}
      {totalReturnedCount > 0 && currentSide !== 'storyAnalysis' ? (
        <Link
          className="blackboard-chip blackboard-chip-link"
          to={studioRoutePaths.storyAnalysis}
          state={storyAnalysisLinkState}
          title="回到解析查看本轮回传历史"
        >
          累计回传 {totalReturnedCount}
        </Link>
      ) : (
        <span className="blackboard-chip">累计回传 {totalReturnedCount}</span>
      )}
      <span
        className={
          canCloseRound
            ? 'blackboard-chip blackboard-chip-ready'
            : allCleared
              ? 'blackboard-chip'
              : 'blackboard-chip blackboard-chip-warn'
        }
      >
        {canCloseRound ? '可收口本轮' : allCleared ? '尚无回传' : '协同处理中'}
      </span>
    </div>
  )
}
