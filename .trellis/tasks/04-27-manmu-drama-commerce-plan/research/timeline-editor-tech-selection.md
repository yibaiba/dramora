# Timeline Editor Technical Selection

## Purpose

Choose the first technical direction for Manmu's online timeline editor and export pipeline.

Manmu does not need a Premiere-class editor in v1. It needs a reliable AI manju completion editor: generated shots, keyframes, TTS, subtitles, transitions, preview, replacement/regeneration, and server-side export.

## Reference comparison

| Reference | Strengths | Risks / limits | Manmu decision |
| --- | --- | --- | --- |
| FreeCut | MIT, React/TS, Vite, WebGPU/WebCodecs, multi-track timeline, media library, TTS, subtitles, feature-domain architecture, strict boundary checks. | Very broad editor scope; Chromium-heavy browser APIs; too large to clone for MVP. | Borrow architecture boundaries and feature split; do not copy full scope. |
| VNVE | MIT, visual-novel-oriented, text-first, scene templates, PixiJS + WebCodecs, programmatic scenes. | Less suited for multi-track pro timeline; browser export limitations. | Borrow scene/template model for dialogue-heavy manju scenes. |
| Twick | Modular SDK: timeline, canvas, live-player, browser-render, render-server, cloud export; production guidance favors server export for SaaS. | Sustainable Use License; direct SDK use needs license review. | Borrow modular package boundaries and browser/server export decision tree; avoid direct dependency until license is approved. |
| DesignCombo React Video Editor | Polished React UI, timeline, effects, transitions, preview, Remotion integration. | README shows copyright, no permissive reuse signal; dependency surface is large. | Use only as UI/interaction inspiration unless license is cleared. |
| Remotion | React-based programmatic video composition, strong server rendering ecosystem. | Special license; company/commercial use may require license. Node rendering service adds stack complexity. | Keep as phase-2 render option after license and ops review. |
| ffmpeg.wasm | MIT, browser-side FFmpeg operations. | Experimental; heavy memory/CPU; not ideal for long/complex exports. | Use only for small local utilities later, not MVP export. |

## Decision

Build a **narrow internal timeline editor** for MVP and export through **server-side FFmpeg/render worker**.

Do not adopt a full third-party editor SDK as the first foundation.

Why:

- Manmu's timeline is shot/asset/workflow-aware, not generic media-only editing.
- The editor must support AI regeneration and asset lineage, which generic editors do not model.
- Direct Twick/DesignCombo/Remotion use has license and product-coupling questions.
- Browser export through WebCodecs/ffmpeg.wasm is not reliable enough for paid SaaS exports across Safari/Firefox/low-memory devices.
- A narrow internal editor can be built around the backend `timeline_tracks` and `timeline_clips` schema already planned.

Recommended MVP:

```text
React Timeline UI
  -> canonical timeline DTO
  -> Go API persists timeline rows
  -> Go/River export job
  -> FFmpeg worker renders MP4
  -> output asset + export row + SSE progress
```

## MVP editor capabilities

Must have:

- track types:
  - video,
  - image/keyframe,
  - audio/TTS,
  - subtitle/text,
  - transition/effect.
- operations:
  - trim,
  - split,
  - move,
  - delete,
  - duplicate,
  - reorder,
  - snap to shot boundaries,
  - replace clip with regenerated asset,
  - preserve clip placement when replacing shot output.
- preview:
  - timeline playhead,
  - selected range playback,
  - video/audio/subtitle sync,
  - poster/thumbnail previews for dense tracks.
- subtitles:
  - edit text,
  - adjust start/end,
  - style preset,
  - burn-in during export.
- TTS:
  - create audio clip per dialogue/subtitle line,
  - replace existing clip while keeping timing.
- export:
  - MP4 1080p/720p presets,
  - progress events,
  - download output asset,
  - error recovery and retry.

Can defer:

- browser-side full export,
- WebGPU effects suite,
- advanced color grading,
- graph keyframe editor,
- nested compositions,
- source monitor,
- waveform editing beyond simple display,
- multi-user real-time editing,
- plugin marketplace.

## Data model

Use backend as source of truth:

```text
Timeline
  id
  project_id
  episode_id
  version_no
  status
  duration_ms

TimelineTrack
  id
  timeline_id
  track_type
  sequence_no
  muted
  locked

TimelineClip
  id
  track_id
  asset_id
  shot_id
  start_ms
  duration_ms
  source_start_ms
  source_duration_ms
  transition_in
  transition_out
  effects
  metadata
```

Frontend local state:

- selected clips,
- playhead,
- zoom,
- drag/resize operation,
- snap settings,
- panel layout,
- preview quality.

Rules:

- Backend timeline rows are canonical.
- Local drag state is committed through explicit save/autosave.
- Use optimistic concurrency with `version_no`.
- Export must reference a specific timeline version.
- Generated shot replacement creates a new clip asset reference, not destructive media overwrite.

## Frontend architecture

Recommended packages/modules:

```text
features/editor-shell
  layout, panels, keyboard shortcuts

features/timeline-model
  timeline DTO normalization, clip calculations, snapping, validation

features/timeline-ui
  track list, ruler, clips, drag/resize interactions

features/preview-player
  HTMLVideoElement/audio/subtitle preview composition for MVP

features/subtitle-editor
  subtitle line list, timing, style preset

features/voice-panel
  TTS generation and audio clip replacement

features/export-panel
  export presets, job progress, download
```

State split:

- TanStack Query: timeline, assets, shots, export jobs.
- Zustand: high-frequency local editor state.
- URL: selected shot/clip, active tab, timeline version.

Implementation note:

- Keep timeline calculation functions framework-agnostic and unit-testable.
- Do not put business status transitions in frontend state.
- Do not couple timeline UI directly to provider/job raw payloads.

## Preview strategy

MVP preview can be simpler than final export:

1. Use generated video/image/audio asset URLs from object storage.
2. Render the active video or image clip in preview.
3. Overlay subtitles with CSS/HTML.
4. Crossfade/slide previews can be approximate.
5. Server export remains the authoritative final render.

Why not full browser composition first:

- multi-track media sync is hard,
- browser codec support is fragmented,
- AI-generated videos can be large,
- paid exports require queueing/retry/progress/audit.

Phase-2 preview:

- canvas-based compositor,
- WebCodecs for local preview/export on Chromium,
- Remotion player for React composition if license is accepted,
- waveform rendering and fine audio sync.

## Export strategy

MVP: Go worker + FFmpeg.

Pipeline:

```text
POST /timelines/{timelineId}/exports
  -> validate timeline version
  -> create export row + generation/export job
  -> River worker downloads or streams source assets
  -> render concat/filter_complex/subtitles
  -> upload output MP4 to object storage
  -> create output asset
  -> mark export succeeded
  -> emit SSE events
```

FFmpeg responsibilities:

- concatenate shot videos/images,
- trim source ranges,
- mix TTS/audio tracks,
- burn subtitles,
- basic fade/crossfade/slide transitions,
- normalize resolution/fps/aspect ratio.

Rules:

- Export worker should use temp workspace per export job.
- Temporary files are cleaned after success/failure.
- Export job records input asset IDs and timeline version for reproducibility.
- Export should fail clearly if a source asset is missing or inaccessible.

Phase 2 options:

- Node/Remotion render worker for React-based motion graphics,
- Twick render-server if license and integration are approved,
- browser WebCodecs export for small projects where supported,
- cloud function/Lambda render for burst scaling.

## Timeline UI interaction details

Required keyboard shortcuts:

- Space: play/pause.
- Left/Right: previous/next frame or small step.
- Ctrl/Cmd+K: split at playhead.
- Delete: delete selected clip.
- Ctrl/Cmd+Z / Shift+Ctrl/Cmd+Z: undo/redo local edit before save.
- `+` / `-`: zoom in/out.

Accessibility alternatives:

- Move clip using inspector numeric fields.
- Change shot status through menu, not drag only.
- Split/delete buttons in toolbar.
- All clip actions available via context menu and keyboard.

Performance:

- Virtualize track rows if many tracks.
- Render only visible time window for clip elements.
- Debounce autosave.
- Avoid rerendering all clips on playhead ticks.
- Use thumbnails/posters, not full video, in clips.

## How this maps to AI manju

Manmu-specific editor features:

- clip can link back to `shot_id`,
- shot replacement keeps timeline placement,
- regenerate selected shot from timeline,
- show continuity/safety issues on clips,
- show cost badge for generated media,
- show provider/model lineage in clip inspector,
- send selected timeline range to Editor Agent for rough-cut suggestions,
- export output asset remains linked to timeline version and input asset lineage.

## Implementation order

1. Timeline domain calculations:
   - duration,
   - overlap detection,
   - split/trim/move,
   - snap,
   - validation.
2. Timeline API integration:
   - load/save timeline,
   - optimistic version conflict handling.
3. UI shell:
   - preview panel,
   - track panel,
   - inspector,
   - toolbar.
4. Basic clip operations:
   - add from approved shot,
   - trim,
   - split,
   - move,
   - delete.
5. Subtitle and TTS clips.
6. Export job submit + progress + download.
7. Regenerate/replace selected shot clip.
8. Basic transitions.

## Final recommendation

For Manmu v1:

- **Build internal timeline editor**.
- **Borrow FreeCut's feature-domain structure**.
- **Borrow VNVE's scene/template concept** for dialogue-heavy manju segments.
- **Borrow Twick's modular split and production export decision tree**, but do not depend on Twick until license is reviewed.
- **Use server-side FFmpeg export first**.
- **Keep Remotion/WebCodecs/ffmpeg.wasm as optional phase-2 render paths**.
