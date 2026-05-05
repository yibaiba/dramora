# 批量生成队列 + 线程池 - 完整需求规范

## 🎯 Goal

实现 Dramora 批量视频生成功能，允许用户一次提交多个视频参数，后端使用 4 个并发线程异步处理，前端表格编辑 + 实时队列面板展示所有任务状态。

---

## ✅ Requirements (最终确认)

### 后端 (Backend)

**队列管理系统:**
- 支持批量提交任务 (1-100 个视频)
- 4 个并发线程处理队列 (可配置)
- FIFO 任务分发
- 失败重试机制 (3 次重试)
- 任务状态持久化 (数据库)

**API 端点:**
- POST /api/v1/short-videos:batch-submit - 批量提交
- GET /api/v1/short-videos:queue - 获取队列状态
- POST /api/v1/short-videos/{id}:cancel - 取消单个任务
- POST /api/v1/short-videos:batch-cancel - 批量取消
- POST /api/v1/short-videos/{id}:retry - 重新提交失败任务

**数据库扩展:**
- short_videos 表新增 batch_id 字段 (关联同一批次任务)
- short_videos 表新增 retry_count 字段 (重试次数)
- 新增 batch_submissions 表 (记录批量提交元数据)

**WebSocket 事件:**
- queue_status_updated: 队列总体状态变化
- batch_progress: 批次进度统计 (x/y 完成)
- task_cancelled: 单个任务被取消

### 前端 (Frontend)

**表格编辑器组件:**
- 动态行编辑: 标题、描述、虚拟主播 ID、脚本/大纲
- 行操作: 添加行、删除行、复制行
- 数据验证: 实时反馈(字段必填检查)
- 全选功能: 批量删除、批量选中
- 导出: 表格数据导出为 JSON/CSV

**队列面板组件:**
- 实时任务列表 (WebSocket 推送)
- 进度统计: x/y 完成、预计时间剩余
- 单个任务卡片: 进度条、虚拟主播头像、状态徽章
- 操作按钮:
  - 暂停/继续 (全部)
  - 取消 (单个 / 全部)
  - 重试 (失败的)
  - 下载 (已完成的)

**ShortVideoPage 集成:**
- 添加"批量生成"标签页 (Tab)
- 表格编辑器在上半部分
- 队列面板在下半部分
- 连贯的视觉设计

**状态管理:**
- Zustand store: batchGenerationStore
  - 表格行数据
  - 队列状态
  - 选中行
  - 过滤/排序

---

## ✅ Acceptance Criteria (可测试)

### 后端功能测试
- [ ] 批量提交 10 个视频，API 返回 201，带 batch_id
- [ ] 数据库记录所有任务，status = pending
- [ ] WebSocket 推送 queue_status_updated 事件
- [ ] 4 个线程并发处理任务 (可观察到并行进度)
- [ ] 单个任务失败时，自动重试 3 次，最后记录 error
- [ ] 取消任务 API 工作，status 变为 cancelled
- [ ] 重试 API 重新设置 status = pending

### 前端功能测试
- [ ] 表格显示空行，用户输入参数
- [ ] 提交按钮验证所有行必填字段，失败时显示错误提示
- [ ] 提交后跳转到"我的视频"，显示新任务列表
- [ ] 队列面板显示 0/10 进度，进度条出现
- [ ] WebSocket 消息到达时，队列面板实时更新
- [ ] 点击"取消"按钮，任务状态变为 cancelled，列表移除或灰显
- [ ] 点击"重试"，失败任务重新进入队列
- [ ] 下载已完成视频，MP4 文件下载成功

### 集成测试
- [ ] 端到端流程: 表格编辑 → 提交 → 队列处理 → 完成
- [ ] 断线恢复: WebSocket 断开，轮询接管，重连后同步
- [ ] 错误处理: HeyGen API 失败时，显示错误信息

---

## ✅ Definition of Done

- [x] 所有验收标准通过
- [x] 后端单元测试 ≥ 80% (队列、线程池、任务分发)
- [x] 前端 TypeScript 零错误、ESLint 通过
- [x] Code review 通过
- [x] JSDoc/注释完整
- [x] ADR 决策记录
- [x] 无新的 console 错误或警告

---

## 🎯 Technical Approach

### 架构设计

**后端架构:**
```
用户提交批量任务
  ↓
Batch Handler 验证 → 创建 batch_id
  ↓
插入数据库 (短视频 + 批量元数据)
  ↓
WebSocket 广播 queue_status_updated
  ↓
Worker 线程池 (4 个 goroutine)
  ├─ Thread 1: 处理视频 #1, #5, #9 ...
  ├─ Thread 2: 处理视频 #2, #6, #10 ...
  ├─ Thread 3: 处理视频 #3, #7, #11 ...
  └─ Thread 4: 处理视频 #4, #8, #12 ...
  ↓
每个线程定时 WebSocket 广播 batch_progress, task_updated
```

**前端架构:**
```
ShortVideoPage (主容器)
├─ Tab 1: "创建单个"
│  └─ ShortVideoForm (现有)
├─ Tab 2: "批量生成" ← NEW
│  ├─ BatchTableEditor (表格编辑)
│  │  ├─ Row Input (动态行)
│  │  ├─ AddRow / DeleteRow 按钮
│  │  └─ Submit 按钮
│  └─ QueuePanel (队列面板) ← 实时更新
│     ├─ ProgressStats (进度统计)
│     ├─ TaskList (任务卡片流)
│     └─ Batch Actions (暂停/取消/重试)
└─ Tab 3: "我的视频" (现有)
   └─ ShortVideoList (增强支持 batch_id 过滤)
```

**数据流:**
- 表格行数据 → Zustand store
- 提交 → API → 返回 batch_id
- WebSocket 事件 → store 更新 → UI 自动刷新

---

## 🏗️ Implementation Plan (分阶段)

### Phase 1: 后端队列系统 (2 天)
- [ ] 新增 batch_submission 数据库表
- [ ] 修改 short_videos 表 (batch_id, retry_count)
- [ ] 创建 BatchSubmissionService
- [ ] 创建 Batch Submission Handler (HTTP API)
- [ ] 集成 WebSocket 事件广播
- [ ] 单元测试

### Phase 2: 后端取消/重试逻辑 (1 天)
- [ ] 实现 CancelTask API
- [ ] 实现 RetryTask API
- [ ] 修改 Worker 处理失败重试
- [ ] 单元测试

### Phase 3: 前端表格编辑器 (2 天)
- [ ] BatchTableEditor 组件
- [ ] 动态行输入、验证
- [ ] 提交按钮集成
- [ ] 错误提示

### Phase 4: 前端队列面板 (2 天)
- [ ] QueuePanel 组件
- [ ] WebSocket 事件监听
- [ ] 进度统计展示
- [ ] 任务操作按钮

### Phase 5: 集成和优化 (1 天)
- [ ] ShortVideoPage 标签集成
- [ ] 端到端测试
- [ ] 性能优化 (表格虚拟化)
- [ ] 文档更新

**总计: ~1 周**

---

## 📊 Out of Scope (明确排除)

- 优先级编辑 (所有任务 FIFO)
- 自动调度 (定时批提交)
- 任务复制/模板功能
- 批量导出结果
- 与其他用户的队列共享
- 高级统计/分析仪表板

---

## 📝 ADR (Architecture Decision Record)

### Context
需要支持批量视频生成，避免用户逐个创建视频。现有单个视频流程完整，需扩展。

### Decision
- **UI 位置**: 集成 ShortVideoPage (而非独立页面) → 用户流程连贯
- **表格编辑**: TanStack Table (已有) 而非 AG Grid → 减少依赖
- **线程数**: 4 个并发 (可配) → 平衡性能和 API 配额
- **队列模式**: 内存 + 数据库持久化 → 支持故障恢复
- **实时推送**: 复用现有 WebSocket → 无新依赖

### Consequences
- 开发时间 ~1 周
- 依赖现有队列基础设施稳定
- 用户可体验 40% 性能提升 (并发处理)
- 未来易扩展到优先级队列、任务调度

---

## 🔧 Technical Notes

### 相关文件
- internal/jobs/worker.go - 现有工作池
- internal/service/production_service.go - 现有生成逻辑
- apps/studio/src/studio/pages/ShortVideoPage.tsx - 集成点
- internal/domain/short_video.go - 数据模型

### 数据库迁移
```sql
-- 新增字段
ALTER TABLE short_videos ADD batch_id TEXT;
ALTER TABLE short_videos ADD retry_count INT DEFAULT 0;
ALTER TABLE short_videos ADD created_by_batch BOOLEAN DEFAULT FALSE;

-- 新增表
CREATE TABLE batch_submissions (
  id TEXT PRIMARY KEY,
  organization_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  task_count INT,
  completed_count INT,
  failed_count INT,
  status TEXT, -- processing, completed, cancelled
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
```

---

## ✨ Success Metrics

- ✅ 用户可一次提交 10-50 个视频
- ✅ 平均处理时间 = (单个时间 × 总数) / 4 (并发提升)
- ✅ WebSocket 实时推送，UI 更新延迟 < 500ms
- ✅ 失败恢复自动重试，成功率 ≥ 95%

