# WebSocket 实时更新 - 替代轮询

## 🎯 Goal

将 HeyGen 视频生成的状态推送从 HTTP 轮询（2 秒间隔）改为 WebSocket 实时推送，减少网络消耗，降低更新延时，提升用户体验。

目标延时：从 < 2s 降至 < 100ms

---

## ✅ Requirements

### 后端实现

- [ ] 集成 socket.io 库（`github.com/socketio/socket.io-go`）
- [ ] 创建 `/ws` 连接端点
- [ ] 认证：验证用户身份，仅推送该用户的视频更新
- [ ] 事件推送：中频率（每 0.5-1 秒）推送视频生成进度
- [ ] 消息格式：复用 `internal/realtime.Event` 结构
- [ ] 生成任务完成时立即推送 `EventGenerationCompleted` 事件
- [ ] 生成任务失败时推送 `EventGenerationFailed` 事件

### 前端实现

- [ ] 集成 socket.io-client 库
- [ ] 创建 WebSocket 客户端 hook（自动连接、重连、心跳）
- [ ] 接收 `generation.progress` 事件 → 更新 React Query 缓存
- [ ] 接收 `generation.completed` 事件 → 标记完成、显示缩略图
- [ ] 接收 `generation.failed` 事件 → 显示错误信息
- [ ] 修改 ShortVideoList：停用 HTTP 轮询，改用 WebSocket 消息
- [ ] 优雅降级：若 WS 连接丢失，自动回退轮询

### 集成与验证

- [ ] 端到端测试：生成视频 → 观察实时进度更新
- [ ] 连接断开测试：模拟网络中断 → 自动重连 → 恢复更新
- [ ] 延时测试：验证更新延时 < 100ms
- [ ] 负载测试：多个用户同时生成 → 性能正常

---

## ✅ Acceptance Criteria

- [ ] 视频生成状态更新延时 < 100ms（从任务完成到前端收到消息）
- [ ] WebSocket 连接断开后自动重连（指数退避）
- [ ] 仅向认证用户推送该用户的视频更新（没有信息泄露）
- [ ] 前端收到事件消息后，ShortVideoList 界面自动更新（无需刷新页面）
- [ ] 浏览器关闭 DevTools Network 模拟离线 5 秒后，连接自动恢复并继续更新
- [ ] 所有测试通过（后端单元测试）
- [ ] TypeScript 编译零错误，ESLint 无新增错误

---

## 📋 Decision Log

| 决策 | 选项 | 原因 |
|------|------|------|
| 推送频率 | 中频（0.5-1 秒） | 平衡实时性和网络消耗，允许动态进度条显示 |
| WebSocket 库 | socket.io | 开箱即用的重连、心跳、降级；事件模式匹配现有 realtime 框架 |
| 功能范围 | 仅视频生成推送 | 聚焦现有需求，快速交付；后续其他功能另行评估 |
| 兼容性 | 现代浏览器（Chrome/Firefox/Safari） | 无需支持 IE11；移动端优先 |

---

## 🏗️ Technical Approach

### 后端架构

```
ProductionService.StartShortVideoGeneration
  └─ 异步任务运行（goroutine）
       └─ 定时轮询 HeyGen 进度（每 0.5-1 秒）
            └─ 更新数据库 + 发送 realtime.Event
                 └─ 广播到所有连接的 WebSocket 客户端（该用户）
```

### 前端架构

```
ShortVideoPage
  └─ useWebSocketEvents() hook
       └─ socket.io 连接管理
            └─ 监听 generation.progress, generation.completed, generation.failed
                 └─ 更新 React Query 缓存
                      └─ ShortVideoList 自动更新（通过 useShortVideos）
```

### 消息格式

```json
{
  "type": "generation.progress",
  "data": {
    "videoId": "550e8400-e29b-41d4-a716-446655440000",
    "status": "generating",
    "progress": 45,
    "estimatedSecondsRemaining": 15
  }
}
```

---

## 📝 Implementation Plan (4 PRs)

### PR1: 后端 WebSocket 服务器 + 事件推送

**文件**:
- `internal/httpapi/websocket_handler.go` (新建，~200 行)
- `internal/httpapi/router.go` (修改，添加 WS 端点)
- `internal/service/production_service.go` (修改，添加事件发送)
- `go.mod` (添加 socket.io 依赖)

**工作**:
- 集成 socket.io 库
- 实现 `/ws` 连接端点（带认证）
- 在 StartShortVideoGeneration 中定时发送事件

**验证**:
- 后端编译成功
- 单元测试全通过
- 手动测试：客户端能接收事件

---

### PR2: 前端 WebSocket 客户端 + Hooks

**文件**:
- `apps/studio/src/api/websocket.ts` (新建，~150 行)
- `apps/studio/src/api/hooks.ts` (修改，添加 useWebSocketEvents)
- `package.json` (添加 socket.io-client 依赖)

**工作**:
- 集成 socket.io-client
- 创建 useWebSocketEvents 并自动重连
- 监听 generation.* 事件 → 更新 React Query 缓存

**验证**:
- 前端编译成功
- TypeScript 零错误
- 手动测试：连接建立，能接收事件

---

### PR3: 集成到 ShortVideoList

**文件**:
- `apps/studio/src/studio/components/ShortVideoList.tsx` (修改)
- `apps/studio/src/api/hooks.ts` (修改，停用轮询)

**工作**:
- 调用 useWebSocketEvents hook
- 移除 useShortVideos 的 refetchInterval（停用轮询）
- 添加回退轮询（若 WS 连接失败）

**验证**:
- 前端构建成功
- 手动测试：创建视频 → 观察实时更新
- 模拟网络断开 → 观察自动回退轮询

---

### PR4: 性能测试 + 文档

**文件**:
- `docs/websocket-implementation.md` (新建)
- `internal/httpapi/websocket_handler_test.go` (新建，~100 行)

**工作**:
- 编写集成测试
- 验证延时、负载、断线恢复
- 文档：使用说明、故障排除

**验证**:
- 所有测试通过
- 负载测试：10 并发用户，无性能下降

---

## ⚠️ Out of Scope

- [ ] 其他功能的实时推送（通知、消息、工作流等）
- [ ] 消息持久化、已读状态
- [ ] WebSocket 负载均衡（多实例共享连接）
- [ ] 移动 App WebSocket（仅 Web 浏览器）

---

## 📚 Technical Notes

### 现有资源

- `internal/realtime/events.go` - 已有事件定义：EventGenerationProgress, EventGenerationCompleted, EventGenerationFailed
- `internal/service/production_service.go` - 异步任务运行的地方

### 关键决策

1. **认证方式**: 使用 JWT token（现有认证机制）
2. **消息编码**: JSON（简单可读）
3. **心跳间隔**: socket.io 默认 25 秒（可配置）
4. **重连策略**: socket.io 指数退避（1s, 2s, 4s, ... max 60s）

### 可能的问题

1. **Goroutine 生命周期** - 生成任务运行在 goroutine 中，任务结束后如何通知 WS？
   - 方案：使用 channel 或回调函数
2. **多实例部署** - 若后端有多个实例，消息如何广播？
   - 方案：暂不考虑（MVP），future 可用 Redis Pub/Sub

---

## 🚀 Ready for Implementation

所有决策已确认，可以开始实现！

**预计工期**: 2-3 天（PR1-4）
**风险**: 低（socket.io 生产级库，标准模式）

