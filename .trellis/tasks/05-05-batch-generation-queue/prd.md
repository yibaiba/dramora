# brainstorm: 批量生成队列 + 线程池 + UI

## Goal

在 Dramora 中实现批量视频生成功能，允许用户一次提交多个视频生成任务，后端使用线程池管理队列并并行处理，前端显示队列进度和单个视频状态。

## What I already know

### 现有功能基础
* WebSocket 实时更新已完成 (PR1-2)
* 单个视频生成流程已完成 (HeyGen 集成 + 轮询)
* React Query 状态管理已就位
* ShortVideoPage 已有基础 UI

### 可能的架构参考
* 项目使用内联 worker (MANMU_INLINE_WORKER=true)
* 后端有 Go 中的任务队列系统 (internal/jobs/)
* 前端有 Zustand 状态管理用于复杂场景

## Assumptions (temporary - 待验证)

* 批量提交支持 1-100 个视频
* 后端使用 worker 线程池处理队列
* 前端显示队列面板 + 单个视频进度
* 优先级/暂停/恢复等高级功能可能超出 MVP 范围

## Open Questions (待解答)

* [用户输入待定] 批量生成的优先级是什么？
* [用户输入待定] 需要什么样的队列面板 UI？
* [用户输入待定] 是否支持暂停/恢复/取消单个任务？

## Requirements (evolving)

* 支持一次提交多个视频生成任务
* 后端队列管理 (FIFO 或优先级)
* 线程池并发处理 (configurable 线程数)
* WebSocket 实时推送队列状态
* 前端队列面板显示

## Acceptance Criteria (evolving)

* [ ] 用户可创建批量视频生成任务
* [ ] 队列面板显示所有待处理/进行中/完成视频
* [ ] WebSocket 实时更新队列状态
* [ ] 可查看单个视频详细进度

## Definition of Done (team quality bar)

* 后端: 单元测试 (队列、线程池、任务分发)
* 前端: TypeScript 零错误，ESLint 通过
* 集成: 端到端流程测试
* 文档: 注释 + ADR 决策记录

## Out of Scope (explicit)

* 任务优先级编辑 (MVP 是 FIFO)
* 任务暂停/恢复
* 持久化队列 (仅内存)
* 任务重试机制
* 批量操作撤销/恢复

## Technical Notes

### 代码查询
* 需查看 internal/jobs/ 现有队列设计
* 需查看 internal/provider/heygen/ 生成逻辑
* 需查看 internal/app/inline_worker.go 工作池实现

### 性能考虑
* 单个视频生成时间: 30-120 秒 (取决于 HeyGen)
* 默认线程数: 2-4 并发 (避免 API 配额问题)
* WebSocket 消息频率: 每个进度更新推送一条

### 文件清单 (待补充)
* 后端: internal/service/batch_*.go (新)
* 后端: internal/httpapi/batch_*.go (新)
* 前端: src/components/BatchGenerationPanel.tsx (新)
* 前端: src/hooks/useBatchGeneration.ts (新)

## Research Progress

(待进行技术研究)

### 批量处理参考
- [ ] 查看现有队列模式 (internal/jobs/)
- [ ] 查看工作池设计 (inline_worker.go)
- [ ] 查看 WebSocket 消息定义 (types/websocket.ts)

### UI/UX 参考
- [ ] 查看类似工具的批量面板 (FFmpeg GUI、视频编辑软件)
- [ ] 规划进度显示 (列表/卡片/图表)


## Decisions Made

### Decision 1: UI 位置
**选择**: ShortVideoPage 内集成  
**理由**: 用户流程连贯，允许快速从单个到批量提交

---


### Decision 2: 并发线程数
**选择**: 均衡 - 4 个并发  
**理由**: 生产环境推荐，平衡性能和稳定性

---


### Decision 3: 批量输入方式
**选择**: 表格编辑器  
**理由**: 直观、即时验证、支持中等数量任务 (10-50 个)
**工具选项**: TanStack Table (已在项目中) 或 AG Grid (外部库)

---

