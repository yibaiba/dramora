# 批量生成队列 - 实现指南

## 📋 工作分解

### 并行工作流 (Phase 1-2)

#### 后端 (Backend Team)
**Phase 1: 数据库 + 服务层**
- [ ] 创建数据库迁移 (batch_id, retry_count, batch_submissions 表)
- [ ] 扩展 ShortVideo domain 模型
- [ ] 创建 BatchSubmissionService
- [ ] 实现队列查询逻辑

**Phase 2: HTTP API + WebSocket**
- [ ] 创建 BatchSubmissionHandler
- [ ] 实现 5 个 API 端点 (submit/queue/cancel/batch-cancel/retry)
- [ ] 集成 WebSocket 事件广播
- [ ] 单元测试 (TestBatchSubmission)

#### 前端 (Frontend Team)
**Phase 3: 组件开发**
- [ ] BatchTableEditor 组件 (TanStack Table)
- [ ] QueuePanel 组件
- [ ] Zustand store (batchGenerationStore)
- [ ] 集成到 ShortVideoPage

### 集成 (Phase 5)
- [ ] 端到端测试 (表格 → 队列 → 完成)
- [ ] WebSocket 连接性测试
- [ ] 性能优化

---

## 🎯 关键文件位置

### 后端
```
internal/domain/short_video.go           ← 扩展 ShortVideo 模型
internal/repo/short_video_repo.go        ← 新增 GetQueue, CancelTask 等
internal/service/batch_service.go        ← 新建 (批量提交业务逻辑)
internal/httpapi/batch_handler.go        ← 新建 (HTTP 处理器)
internal/app/container.go                ← 注入 BatchService
db/migrations/000034_batch_system.go     ← 新建 (迁移)
```

### 前端
```
apps/studio/src/types/batch.ts           ← 新建 (类型定义)
apps/studio/src/components/BatchTableEditor.tsx   ← 新建
apps/studio/src/components/QueuePanel.tsx         ← 新建
apps/studio/src/store/batchGenerationStore.ts     ← 新建
apps/studio/src/studio/pages/ShortVideoPage.tsx   ← 修改 (标签集成)
apps/studio/src/api/batch.ts             ← 新建 (API 客户端)
apps/studio/src/api/hooks.ts             ← 新增 hook
```

---

## 🔄 开发流程

### 后端优先启动

1. **创建迁移** (15 min)
   ```bash
   # 在 db/migrations/ 创建新文件
   # 添加 batch_id, retry_count 字段
   # 创建 batch_submissions 表
   ```

2. **更新 Domain** (30 min)
   ```go
   // internal/domain/short_video.go
   type ShortVideo struct {
       // ... 现有字段
       BatchID string      // 关联批次 ID
       RetryCount int      // 重试次数
       CreatedByBatch bool // 是否来自批量提交
   }
   ```

3. **创建 Service** (2 hours)
   ```go
   // internal/service/batch_service.go
   type BatchSubmissionService struct {
       repo     *repo.ShortVideoRepo
       producer jobs.Producer
       ws       *websocket.Manager
   }
   
   func (s *BatchSubmissionService) SubmitBatch(ctx context.Context, tasks []CreateShortVideoRequest) (string, error)
   func (s *BatchSubmissionService) GetQueueStatus(ctx context.Context) (*QueueStatus, error)
   func (s *BatchSubmissionService) CancelTask(ctx context.Context, videoID string) error
   ```

4. **创建 Handler** (1.5 hours)
   ```go
   // internal/httpapi/batch_handler.go
   type BatchHandler struct {
       service *service.BatchSubmissionService
   }
   
   func (h *BatchHandler) HandleSubmitBatch(w http.ResponseWriter, r *http.Request)
   func (h *BatchHandler) HandleGetQueue(w http.ResponseWriter, r *http.Request)
   func (h *BatchHandler) HandleCancelTask(w http.ResponseWriter, r *http.Request)
   ```

5. **集成路由** (30 min)
   ```go
   // internal/httpapi/router.go
   router.Post("/short-videos:batch-submit", h.batchHandler.HandleSubmitBatch)
   router.Get("/short-videos:queue", h.batchHandler.HandleGetQueue)
   router.Post("/short-videos/{id}:cancel", h.batchHandler.HandleCancelTask)
   ```

6. **单元测试** (1 hour)
   ```go
   // internal/service/batch_service_test.go
   func TestSubmitBatch_Success(t *testing.T)
   func TestCancelTask_Success(t *testing.T)
   func TestRetryTask_Success(t *testing.T)
   ```

**后端总计: ~5-6 hours** ✅

### 前端并行启动

1. **类型定义** (30 min)
   ```typescript
   // apps/studio/src/types/batch.ts
   export interface BatchTask {
       id: string
       batchId: string
       title: string
       description: string
       heyGenAvatarId: HeyGenAvatarId
       status: 'pending' | 'generating' | 'completed' | 'failed' | 'cancelled'
       progress: number
   }
   ```

2. **表格编辑器** (3 hours)
   ```typescript
   // BatchTableEditor.tsx
   - 使用 TanStack Table
   - 动态行 (Add/Delete/Copy)
   - 实时验证
   - Submit 按钮
   ```

3. **队列面板** (2 hours)
   ```typescript
   // QueuePanel.tsx
   - WebSocket 监听 queue_status_updated
   - 进度统计展示
   - 任务卡片列表
   - 操作按钮 (取消/重试)
   ```

4. **Store** (1 hour)
   ```typescript
   // batchGenerationStore.ts
   - 表格行数据
   - 队列状态
   - 选中行
   - 过滤/排序
   ```

5. **API 客户端** (1 hour)
   ```typescript
   // apps/studio/src/api/batch.ts
   export async function submitBatchVideos(tasks: CreateVideoRequest[]): Promise<SubmitResponse>
   export async function getQueueStatus(): Promise<QueueStatus>
   export async function cancelTask(videoId: string): Promise<void>
   ```

6. **ShortVideoPage 集成** (1 hour)
   ```typescript
   // 添加"批量生成"标签页
   // 嵌入 BatchTableEditor + QueuePanel
   ```

**前端总计: ~8-9 hours** ✅

### 集成与测试 (1-2 hours)
- 端到端流程验证
- WebSocket 连接性
- 错误处理
- 性能测试

---

## ✅ 验收标准速查

### 快速检查清单

**后端完成标志:**
- [ ] `go test ./...` 通过 (包含新测试)
- [ ] `go build ./...` 成功
- [ ] API 可返回 batch_id (201 Created)
- [ ] 数据库有新表

**前端完成标志:**
- [ ] `npm run build` 成功 (Vite 无错误)
- [ ] `npm run lint` 通过
- [ ] TypeScript 零错误
- [ ] 表格和队列面板显示

**集成完成标志:**
- [ ] 浏览器访问 ShortVideoPage → 看到"批量生成"标签
- [ ] 输入表格数据 → 点击提交
- [ ] 看到队列面板实时更新
- [ ] WebSocket 连接正常 (Console 看到日志)

---

## 🚀 启动指令

```bash
# 后端: 启动任务上下文
cd /Users/yibai/Code/GolandProjects/dramora
python3 ./.trellis/scripts/task.py start .trellis/tasks/05-05-batch-generation-queue

# 前端: 启动开发服务器
cd apps/studio
npm run dev

# 后端: 启动 API
MANMU_INLINE_WORKER=true go run apps/api/main.go
```

---

## 📞 卡顿排查

| 问题 | 可能原因 | 解决方案 |
|------|--------|--------|
| API 返回 404 | 路由未注册 | 检查 router.go 中的路由定义 |
| 前端表格不显示 | 组件导入错误 | 检查 import 路径，确认文件存在 |
| WebSocket 消息未推送 | 事件未触发 | 检查 BatchService 是否调用了广播 |
| 数据库迁移失败 | SQL 语法错误 | 查看迁移日志，检查 schema |
| 测试失败 | Mock 数据不匹配 | 更新 test fixture 以匹配新字段 |

---

## 📚 相关文档

- WebSocket 实现: `.trellis/tasks/05-04-websocket-realtime/prd.md`
- 项目规范: `.trellis/spec/`
- API 约定: `.trellis/spec/backend/directory-structure.md`

---

## 🎯 下一步

当 Phase 1-2 完成后，请运行:

```bash
python3 ./.trellis/scripts/task.py complete .trellis/tasks/05-05-batch-generation-queue
```

