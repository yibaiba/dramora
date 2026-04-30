import {
  Eye,
  Film,
  Lock,
  Maximize2,
  Music2,
  Play,
  Plus,
  Scissors,
  Subtitles,
} from 'lucide-react'
import { useExport, useExportRecovery, useSaveEpisodeTimeline, useStartEpisodeExport } from '../../api/hooks'
import type { Episode, Timeline } from '../../api/types'
import type { StudioShot } from '../types'
import { RecoveryPanel } from './RecoveryPanel'
import {
  buildTimelineRequest,
  exportStatusLabel,
  formatDuration,
  formatTimecode,
  subtitleForShot,
} from '../utils'

type TimelineWorkspaceProps = {
  activeEpisode?: Episode
  displayShots: StudioShot[]
  onAddLocalShot?: () => void
  timeline?: Timeline
  timelineSource: 'saved' | 'storyboard'
}

export function TimelineWorkspace({
  activeEpisode,
  displayShots,
  onAddLocalShot,
  timeline,
  timelineSource,
}: TimelineWorkspaceProps) {
  const saveTimeline = useSaveEpisodeTimeline()
  const startExport = useStartEpisodeExport()
  const exportQuery = useExport(startExport.data?.id)
  const activeExport = exportQuery.data ?? startExport.data
  const exportRecovery = useExportRecovery(activeExport?.id)
  const duration = displayShots.reduce((total, shot) => total + shot.durationMS, 0)

  const saveDraft = () => {
    if (!activeEpisode) return
    saveTimeline.mutate({
      episodeId: activeEpisode.id,
      request: buildTimelineRequest(displayShots),
    })
  }

  return (
    <section className="timeline-dock" aria-labelledby="timeline-title">
      <div className="timeline-toolbar">
        <h2 id="timeline-title">剪辑时间线</h2>
        <div className="transport-controls" aria-label="播放控制">
          <button aria-label="切开片段" type="button">
            <Scissors aria-hidden="true" />
          </button>
          <button aria-label="播放预览" type="button">
            <Play aria-hidden="true" />
          </button>
          <button aria-label="全屏预览" type="button">
            <Maximize2 aria-hidden="true" />
          </button>
        </div>
        <span className="timecode">00:00:11:12</span>
        <span className="timeline-source-pill">
          {timelineSource === 'saved'
            ? `已保存 Timeline v${timeline?.version ?? 1}`
            : 'Storyboard 派生草稿'}
        </span>
        <button
          className="ghost-action"
          disabled={!activeEpisode || saveTimeline.isPending}
          onClick={saveDraft}
          type="button"
        >
          保存剪辑
        </button>
        <button
          className="ghost-action"
          disabled={!activeEpisode || !timeline || startExport.isPending}
          onClick={() => activeEpisode && startExport.mutate(activeEpisode.id)}
          type="button"
        >
          开始导出
        </button>
      </div>
      <TimelineRuler />
      <TimelineTracks displayShots={displayShots} onAddLocalShot={onAddLocalShot} />
      <footer className="timeline-footer">
        <span>总时长 {formatTimecode(duration)}</span>
        <span>
          {displayShots.length} 镜 · {Math.round(duration / 1000)} 秒
        </span>
        <span>导出预设 1080p · H.264 · 24fps</span>
        <span>导出状态 {activeExport ? exportStatusLabel(activeExport.status) : '可预览'}</span>
      </footer>
      {activeExport ? (
        <RecoveryPanel
          title="导出任务恢复"
          subtitle={`Export ${activeExport.id.slice(0, 8)} · ${exportStatusLabel(activeExport.status)}`}
          isLoading={exportRecovery.isLoading}
          isError={exportRecovery.isError}
          status={exportRecovery.data?.summary.current_status}
          isTerminal={exportRecovery.data?.summary.is_terminal}
          isRecoverable={exportRecovery.data?.summary.is_recoverable}
          statusEnteredAt={exportRecovery.data?.summary.status_entered_at}
          lastEventAt={exportRecovery.data?.summary.last_event_at}
          totalEventCount={exportRecovery.data?.summary.total_event_count}
          nextHint={exportRecovery.data?.summary.next_hint}
          events={exportRecovery.data?.events.map((event) => ({
            status: event.status,
            message: event.message,
            created_at: event.created_at,
          }))}
        />
      ) : null}
    </section>
  )
}

function TimelineRuler() {
  return (
    <div className="timeline-ruler" aria-hidden="true">
      {['00:00:00', '00:00:05', '00:00:10', '00:00:15', '00:00:20', '00:00:25', '00:00:30'].map(
        (mark) => (
          <span key={mark}>{mark}</span>
        ),
      )}
    </div>
  )
}

function TimelineTracks({
  displayShots,
  onAddLocalShot,
}: {
  displayShots: StudioShot[]
  onAddLocalShot?: () => void
}) {
  return (
    <div className="timeline-tracks" aria-label="剪辑轨道">
      <TrackLabel icon={Film} label="V1" name="画面" />
      <div className="timeline-strip video-strip">
        {displayShots.map((shot) => (
          <TimelineClip key={shot.key} shot={shot} />
        ))}
        {onAddLocalShot ? (
          <button className="add-clip" onClick={onAddLocalShot} type="button">
            <Plus aria-hidden="true" /> 添加片段
          </button>
        ) : null}
      </div>
      <TrackLabel icon={Music2} label="A1" name="配乐" />
      <div className="audio-wave">BGM_九霄之上_Main Theme.wav</div>
      <TrackLabel icon={Subtitles} label="S1" name="字幕" />
      <div className="subtitle-strip">
        {displayShots.map((shot) => (
          <span key={shot.key}>{subtitleForShot(shot.code)}</span>
        ))}
      </div>
      <TrackLabel icon={Scissors} label="T1" name="转场" />
      <div className="transition-strip">
        <span>云雾叠化 00:00:15</span>
        <span>雷光闪切 00:00:10</span>
      </div>
    </div>
  )
}

function TrackLabel({
  icon: Icon,
  label,
  name,
}: {
  icon: typeof Film
  label: string
  name: string
}) {
  return (
    <div className="track-label">
      <span>{label}</span>
      <Icon aria-hidden="true" />
      <strong>{name}</strong>
      <Lock aria-hidden="true" />
      <Eye aria-hidden="true" />
    </div>
  )
}

function TimelineClip({ shot }: { shot: StudioShot }) {
  return (
    <article className="timeline-clip">
      <span className={`clip-thumb ${shot.thumbnail}`} aria-hidden="true" />
      <strong>{shot.code}</strong>
      <small>{formatDuration(shot.durationMS)}</small>
    </article>
  )
}
