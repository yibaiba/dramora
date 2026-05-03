# Web 剪辑 - 参考 freecut 架构自建

## Goal

为 Dramora Studio 添加浏览器端视频编辑功能。参考开源项目 freecut 的架构，用 ffmpeg.js + 自建 React 编辑 UI，让用户在 Web 中对**单个视频**和**完整故事时间线**进行编辑、剪辑、特效处理，然后导出为多种格式（MP4、FCPXML、Premiere、DaVinci Resolve）。

**核心目标**: 零成本、高质量、与 Dramora 风格一致的 Web 视频编辑

## Requirements

### 核心功能
1. **Gallery 快速编辑**
   - 在 GalleryPage 中，点击已生成的视频卡片
   - 打开快速编辑面板（侧板或全屏）
   - 支持基础编辑（裁剪、转场、速度调整）

2. **Timeline 完整编辑**
   - `/timeline-export` 页面集成编辑器
   - 编辑完整的故事时间线（多 shot、音乐、字幕）
   - 支持多轨道操作（视频、音频、字幕）

3. **多格式导出**
   - MP4（Web 标准）
   - FCPXML（Final Cut Pro）
   - Premiere 格式（.prproj）
   - DaVinci Resolve 格式（.drp）

### 技术决策
- **编辑方案**: 参考 freecut 架构，自建 React 编辑 UI + ffmpeg.js
- **编辑范围**: 单个视频 + 完整时间线（两者都支持）
- **导出方式**: 多格式导出（不保存回系统作为草稿）
- **成本**: 零成本（完全开源）
- **代码风格**: 保持与 Dramora 一致（Zustand + React Query + Tailwind）

## Acceptance Criteria

- [ ] 用户可在 Gallery 中点击视频打开编辑界面
- [ ] 支持基础编辑（裁剪、转场、速度）
- [ ] 支持导出 MP4（核心）
- [ ] FCPXML 导出支持（已有基础）
- [ ] 编辑界面响应式（支持平板）
- [ ] TypeScript 零错误，ESLint 过检
- [ ] 代码风格与 GalleryPage、QueuePage 一致

## Definition of Done

- TypeScript 零错误
- ESLint 过检
- 前端构建成功 (< 850KB gzip)
- 手动验证：打开编辑、导出 MP4 成功
- 代码审查通过（风格、架构、可维护性）

## Out of Scope (explicit)

- 编辑结果保存回系统（仅导出）
- 实时预览（处理成本高）
- 实时协作编辑
- 离线编辑
- 完整的剪辑软件功能（仅基础编辑）
- GPU 加速渲染

## Technical Approach

### 参考 freecut 架构的关键点

1. **时间线数据模型**
   - Track[] - 视频/音频/字幕轨道
   - Clip[] - 单个片段（有起始时间、持续时间、属性）
   - 编辑操作 (trim, move, split, delete)

2. **编辑状态管理**
   - Zustand store 管理全局编辑状态
   - Playhead 位置跟踪
   - Undo/Redo 支持

3. **导出流程**
   - ffmpeg.js Worker 处理视频转码
   - 进度条 + 预估完成时间
   - 多格式支持（库 or 后端 API）

### 架构设计

```
Timeline/Gallery
  ↓ [点击编辑]
EditVideoModal
  ├─ TimelineCanvas (自建)
  │  ├─ TrackContainer
  │  │  ├─ Track (video/audio/subtitle)
  │  │  ├─ Clip (可拖拽、可调整)
  │  │  └─ Playhead
  │  └─ Ruler (时间标尺)
  ├─ PropertyPanel
  │  ├─ ClipTrim
  │  ├─ SpeedControl
  │  ├─ Transitions
  │  └─ Effects (基础)
  ├─ Toolbar
  │  ├─ Undo/Redo
  │  ├─ Zoom
  │  └─ [导出]
  └─ ExportDialog
     ├─ FormatSelector (MP4/FCPXML/Premiere/DaVinci)
     ├─ QualityOptions
     ├─ ProgressBar
     └─ [开始导出]
```

### 关键库

```json
{
  "@ffmpeg/ffmpeg": "^0.12.x",
  "@ffmpeg/util": "^0.12.x",
  "zustand": "^4.x",
  "mux.js": "^6.x" (可选，用于容器处理)
}
```

### 工作流

1. **编辑状态**
   - Zustand store: EditorStore (tracks, clips, playhead, history)
   - 支持 undo/redo

2. **渲染**
   - Canvas/SVG 绘制时间线 UI
   - 轻量级预览（静态截图或视频第一帧）
   - 完整渲染仅在导出时

3. **导出**
   - ffmpeg.js 在 Web Worker 中处理
   - MP4 输出到浏览器下载
   - FCPXML 通过后端 API（已有 fcpxml-generator）

## Implementation Plan (PRs)

### PR1: 编辑 UI 框架 + 数据模型 (3-4 days)

**文件**:
- `apps/studio/src/studio/components/editor/EditVideoModal.tsx` - 主模态框
- `apps/studio/src/studio/components/editor/TimelineCanvas.tsx` - 时间线画布
- `apps/studio/src/lib/editor/timeline-store.ts` - Zustand store
- `apps/studio/src/lib/editor/types.ts` - 类型定义

**功能**:
- EditVideoModal 脚手架
- TimelineCanvas 基础绘制
- Track、Clip 数据模型
- Playhead、Ruler 组件

**验证**:
- TypeScript 编译成功
- 组件在 GalleryPage 可加载

### PR2: 编辑功能 + MP4 导出 (3-4 days)

**文件**:
- `apps/studio/src/studio/components/editor/PropertyPanel.tsx` - 属性编辑
- `apps/studio/src/lib/editor/ffmpeg-worker.ts` - FFmpeg Web Worker
- `apps/studio/src/lib/editor/export-helpers.ts` - 导出工具

**功能**:
- Clip 裁剪、移动、删除
- 速度调整
- ffmpeg.js 集成
- MP4 导出流程
- 导出进度条

**验证**:
- 编辑功能正常
- MP4 导出成功
- 手动验证导出视频可播放

### PR3: 完整功能 + 多格式导出 (3-4 days)

**文件**:
- `apps/studio/src/studio/components/editor/ExportDialog.tsx` - 导出对话框
- `apps/studio/src/lib/editor/format-converters.ts` - 格式转换
- 修改 GalleryPage、TimelineExportPage

**功能**:
- Timeline 完整编辑（多轨）
- FCPXML 导出（集成 fcpxml-generator）
- Premiere/DaVinci 格式（库 or 后端）
- 在 GalleryPage 添加"编辑"按钮
- 在 TimelineExportPage 集成编辑器

**验证**:
- 完整功能测试
- 多格式导出验证
- GalleryPage 集成成功
- ESLint 和 TypeScript 过检

**总计**: 9-12 days ≈ 1.5-2 周

## Key Decisions (ADR-lite)

**Context**: 
- 需要 Web 视频编辑功能
- OpenVideo SDK 成本考虑
- 发现开源项目 freecut (1.2K stars)

**Decision**: 
- 参考 freecut 架构自建（不 fork）
- 使用 ffmpeg.js + Zustand + 自建 UI
- 保持与 Dramora 代码风格一致

**Consequences**:
- ✅ 零成本、完全定制、长期可维护
- ✅ 与现有代码风格无缝集成
- ✅ 完全理解代码原理
- ⚠️ 工作量 1.5-2 周（vs fork 的 1 周）
- ⚠️ 需要理解 freecut 架构（学习成本）
- 🎯 **值得**: 质量和维护性的投资

**Future**: 
- 如果性能成为瓶颈，可评估 WebGL 加速
- 如果功能复杂度增加，可评估其他库集成

## Research Notes

### freecut 项目信息
- **链接**: https://github.com/walterlow/freecut
- **Stars**: 1.2K
- **技术**: TypeScript, ffmpeg.js, Canvas
- **特点**: 多轨编辑、实时预览、多格式导出

### 研究方向
- 时间线数据模型设计
- ffmpeg.js 命令行参数
- Canvas 绘制优化
- 导出流程 (Worker + Progress)

## Technical Notes

### 依赖安装
```bash
npm install @ffmpeg/ffmpeg @ffmpeg/util zustand
npm install --save-dev @types/ffmpeg.js
```

### 浏览器兼容性
- Chrome/Edge 90+（ffmpeg.js 支持）
- Safari 15+（WebWorkers）
- Firefox 88+（ffmpeg.js 支持）
- 降级方案：仅导出（不编辑）

### 性能考虑
- ffmpeg.js 加载 ~30MB，异步加载 + 缓存
- 浏览器内存：建议 <100MB 视频
- 导出时间：依赖视频长度和浏览器性能

### 测试策略
- 单元测试：编辑操作、状态转换
- 集成测试：编辑 → 导出流程
- 手动测试：不同视频格式、不同长度
- 浏览器测试：Chrome、Safari、Firefox

