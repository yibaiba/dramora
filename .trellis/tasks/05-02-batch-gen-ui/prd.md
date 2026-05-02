# Batch Generation UI - 队列面板与批量生成

## Goal

为 Dramora Studio 实现批量生成队列管理功能，让用户能够在 Storyboard 中批量选择多个分镜（Shot），一键提交生成请求，并在独立队列页面实时跟踪生成进度。

## Key Decisions

✅ **Batch 粒度** = Storyboard Shot 级别
   - 用户在 Storyboard 中多选分镜
   - 一键提交该 Shot 集合的批量生成请求

✅ **UI 架构** = 独立 QueuePage
   - 新建 `/queue` 路由（类似 /gallery）
   - Storyboard 保持简洁，提供快速生成按钮
   - QueuePage 展示全局队列状态

✅ **数据更新** = 轮询（Polling）
   - 每 2 秒调用一次 useGenerationJobs()
   - React Query 自动处理缓存和去重
   - 简单快速，用户体验尚可

## Requirements

**Phase 1: 后端支持**
- [ ] 确保 GET /api/v1/generation-jobs 支持分页/过滤（已有）

**Phase 2: Storyboard 多选 + 批量生成**
- [ ] StoryboardPage 添加多选模式（Shift + 点击 / Ctrl+A）
- [ ] 已选 Shot 计数器
- [ ] 批量生成工具栏（"生成图像"、"生成视频"）
- [ ] 批量提交 job 到后端

**Phase 3: QueuePage（独立页面）**
- [ ] 新建 `/queue` 路由和 QueuePage.tsx
- [ ] 队列卡片汇总（总 jobs、进行中、已完成）
- [ ] 队列表/卡片网格展示 jobs
- [ ] 按状态筛选（queued / rendering / succeeded / failed / canceled）
- [ ] 单 job 取消按钮
- [ ] job 詳情侧板（点击展开）

**Phase 4: 数据更新**
- [ ] useGenerationJobs() 添加 `refetchInterval: 2000`
- [ ] 每个 job 卡片显示更新时间

## Acceptance Criteria

- [ ] 用户可在 Storyboard 中选择多个 Shot（N >= 2）
- [ ] 点击"批量生成"提交 N 个 job
- [ ] QueuePage 页面加载 < 1s
- [ ] job 状态更新延迟 < 3s（轮询间隔）
- [ ] 用户可取消任何 job
- [ ] 点击 job 查看详情（生成模型、耗时、错误信息）
- [ ] 导航菜单显示 Queue 页面（设置为可选，仅当有 jobs 时显示）

## Definition of Done

- TypeScript 零错误
- ESLint 过检
- 前端构建成功（gzip < 700KB）
- 所有 E2E 场景手动验证

## Out of Scope (explicit)

- [ ] 优先级排序/重新排列
- [ ] WebSocket 实时更新（后期升级）
- [ ] 生成参数自定义（用默认参数）
- [ ] 生成结果批量导出
- [ ] 队列持久化/历史记录

## Technical Approach

### Architecture

```
Storyboard (多选 + 工具栏)
  └─> [Batch Submit] → POST /api/v1/episodes/{id}/batch-generate
                      ↓
                 QueuePage (轮询)
                   ├─ QueueStats
                   ├─ QueueFilter
                   └─ JobGrid / JobList
                      └─ JobCard (可取消、可展开详情)
```

### Key Components

1. **StoryboardPage.tsx** (修改)
   - 添加 `selectedShotIds` 本地状态
   - 添加 `isSelectionMode` toggle
   - 添加批量操作工具栏

2. **QueuePage.tsx** (新建)
   - 使用 `useGenerationJobs({ refetchInterval: 2000 })`
   - 展示队列统计
   - 作业列表/网格
   - 作业卡片 + 取消按钮

3. **hooks** (新建 useQueueFilter)
   - 状态过滤逻辑
   - 排序逻辑

### API Integration

**现有**：
- GET /api/v1/generation-jobs（useGenerationJobs）
- 需要检查是否已支持分页

**可能需要新建**：
- POST /api/v1/episodes/{episodeId}/batch-generate (并发生成多个 Shot)
- 或扩展现有 Shot 生成 API 支持批量

## Technical Notes

**Files to modify/create**:
- `apps/studio/src/studio/pages/StoryboardPage.tsx` - 多选 + 工具栏
- `apps/studio/src/studio/pages/QueuePage.tsx` - 新建队列页面
- `apps/studio/src/studio/routes.ts` - 添加 /queue 路由
- `apps/studio/src/App.tsx` - 导入 QueuePage
- `apps/studio/src/api/hooks.ts` - 扩展 useGenerationJobs（轮询）

**Constraints**:
- React Query for server state + refetchInterval
- Tailwind v4 + custom CSS
- Zustand 仅在必要时用于多面板状态
- 响应式设计（支持平板/手机）

**Testing Strategy**:
- 手动：Storyboard 多选 → 生成 → QueuePage 检查
- 验证 job 状态在 2-3s 内更新

## Implementation Plan (small PRs)

**PR1: Storyboard 多选基础**
- 添加多选状态管理
- UI: 多选指示器 + 工具栏
- 测试：能够选择多个 Shot

**PR2: QueuePage MVP**
- 新建 QueuePage + 路由
- 基础队列展示
- 测试：加载队列数据

**PR3: 交互完善**
- job 取消、详情、过滤
- CSS 优化
- 性能调优

## Future Enhancements

- WebSocket 实时更新（当轮询成为瓶颈）
- job 优先级排序
- 批量操作面板（导出、重试）
- 队列历史记录
