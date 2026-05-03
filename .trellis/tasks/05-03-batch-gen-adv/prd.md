# Batch Generation 高级功能：队列管理、重试、优先级

## Goal

增强 Batch Generation 系统的可靠性和可控性，实现：
1. 失败任务的自动或手动重试机制
2. 任务优先级调整（允许提升或降低任务顺序）
3. 队列管理面板（暂停、恢复、清空等操作）

目标是让用户能够更好地管理长时间运行的批量任务，特别是在错误或优先级变化时快速响应。

## What I already know

**现有实现**:
- Batch Generation API 已实现（`POST /api/v1/episodes/{episodeId}/batch-generate`）
- QueuePage 前端已实现（过滤、搜索、取消、实时刷新）
- 后端工作流引擎支持 Job 概念（`internal/jobs/types.go`）
- 数据库中有 GenerationJob 表（包含 status、created_at 等字段）

**现有能力**：
- Job 取消（isCancelable 检查）
- 实时状态同步（2秒刷新）
- 状态映射和标签（16 个不同的 status）

**缺失的能力**：
- 重试逻辑（retry_count、max_retries 字段）
- 优先级管理（priority 字段调整）
- 队列暂停/恢复（worker 级别控制）

## Decisions (锁定) ✅

### 重试策略
**决策**: A) 仅手动重试
- 用户看到失败任务，可以点击"重试"按钮手动重试
- 每次重试是新 job（保留原 job 的失败记录）
- 最多允许 5 次重试（防止滥用）

### 优先级管理
**决策**: B) 精细（0-100 数值滑块）
- 用户拖动滑块或输入数值调整优先级
- 默认优先级: 50（中等）
- 影响执行顺序：数值越高越先执行

### 队列暂停
**决策**: B) 单 Episode 暂停
- 仅暂停当前 episode 的队列处理
- 暂停后创建的新任务仍加入队列（等待恢复）
- Worker 级别停止处理该 episode 的任务

## Requirements (锁定)

* [x] 重试机制 - 手动重试失败的任务（新建 job，最多 5 次）
* [x] 优先级管理 - 0-100 滑块调整任务优先级
* [x] 队列控制 - 单 episode 暂停/恢复处理

## Acceptance Criteria (锁定)

* [ ] 前端 QueuePage 失败任务显示"重试"按钮
* [ ] 前端 QueuePage 所有任务显示优先级滑块（0-100）
* [ ] 前端 QueuePage 显示队列暂停/恢复按钮（Episode 级别）
* [ ] 后端 API POST /api/v1/generation-jobs/{jobId}:retry （返回新 job ID）
* [ ] 后端 API POST /api/v1/generation-jobs/{jobId}:priority （参数: priority 值）
* [ ] 后端 API POST /api/v1/episodes/{episodeId}/queue:pause
* [ ] 后端 API POST /api/v1/episodes/{episodeId}/queue:resume
* [ ] 数据库迁移：添加 retry_count、priority、episode_queue_paused 字段
* [ ] Worker 尊重 episode_queue_paused 状态
* [ ] 测试覆盖：重试创建、优先级排序、暂停状态检查

## Definition of Done (team quality bar)

* Tests added/updated (unit/integration where appropriate)
* `gofmt -w` + `GOTOOLCHAIN=local go test ./...` 全部通过
* TypeScript validation + ESLint 零错误
* 前后端构建成功
* 文档/changelog 更新
* 与现有 QueuePage 功能无缝集成

## Out of Scope (explicit)

* 自动重试或指数退避
* 多级队列（high/normal/low 预设级别）
* 队列持久化到磁盘
* 实时 WebSocket 推送（保留 2 秒轮询）
* 故障自愈（DLQ、死信队列）
* 重试限制的智能调整

## Technical Approach

### 架构决策
1. **重试**: 创建新 GenerationJob，保存父 job ID（追踪重试链）
2. **优先级**: 在 GenerationJob 表添加 priority 字段（默认 50）
3. **暂停**: 在 episodes 表或新 queue_status 表添加 paused_at 字段

### 实现分层
- **数据库**: 迁移添加 retry_count, priority, episode_queue_paused
- **后端服务**: 新增 RetryJob / UpdateJobPriority / PauseEpisodeQueue 方法
- **后端 API**: 3 个新端点（retry、priority、pause/resume）
- **前端 hooks**: useRetryJob / useUpdateJobPriority / usePauseQueue
- **前端 UI**: QueuePage 添加重试按钮、优先级滑块、暂停按钮

### 执行计划（小 PR）

**PR1: 后端基础 - 数据库 + 服务层**
- 数据库迁移：添加字段
- Service 实现：重试、优先级、暂停逻辑
- Worker 更新：排序和尊重暂停状态

**PR2: 后端 API + 前端 hooks**
- 3 个 API 端点注册
- React Query hooks 封装

**PR3: 前端 UI + 测试**
- QueuePage 添加重试按钮、优先级滑块、暂停控制
- 集成测试
- 文档/changelog

## Technical Notes

**文件位置**:
- Backend: `internal/httpapi/batch_generation.go`, `internal/jobs/*`, `internal/service/*`
- Frontend: `apps/studio/src/studio/pages/QueuePage.tsx`, `apps/studio/src/api/`
- Database: `db/migrations/` (需要新增 SQL 迁移)

**关键约定**:
- API: POST 用于操作，无 PUT/PATCH/DELETE
- 认证/授权: 从 PR8 规范检查组织归属
- 约束: Go 1.21 + Chi v5.0.12，TypeScript 零错误

**待调查**:
1. 现有 GenerationJob 表的完整 schema
2. Worker 处理优先级的机制（是否已支持）
3. 暂停状态存储位置（episodes 表 vs 新表）
