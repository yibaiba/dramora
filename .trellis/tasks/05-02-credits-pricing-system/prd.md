# 13 积分计费体系 MVP

## Goal

完成 Dramora 项目的积分（credits）计费体系，从现有的基础钱包系统（余额管理、流水记录）扩展到**完整的商用定价规则**：支持按操作类型定价、扣费逻辑、计费模型（后付制 + 成功才扣费）、以及用户查询界面。

## What I already know

**现有钱包基础设施**（已完成）：
- `internal/domain/wallet.go`: 钱包领域模型（Wallet、WalletTransaction）
- `internal/service/wallet_service.go`: 钱包服务（GetWallet、ListTransactions）
- `internal/repo/wallet_repo.go`: 三层 repo 实现（Memory/SQLite/Postgres）
- `internal/httpapi/wallet.go`: HTTP 路由 + DTO (GET /wallet, POST /wallet/apply)
- `db/migrations`: Wallet & WalletTransaction 表
- 通知集成：wallet_credit/wallet_debit 事件已接入 NotificationService

**现有限制/特点**：
- WalletTransaction.Kind: credit / debit / refund / adjust （分类已定）
- 余额检查已实现（ErrInsufficientBalance）
- 组织隔离已实现（RequestAuthContext）

**前端现状**：
- 暂无 Wallet UI 页面（需从零创建或集成到现有页面）

**用户决策（刚确认）**：
- ✅ 计费时机：**后付制 + 成功才扣费**
  - 故事分析（story_analysis）—— 任务成功完成后扣费
  - 图像生成（image_generation）—— 生成成功后扣费
  - 视频生成（video_generation）—— 生成成功后扣费
  - 故事地图更新、角色编辑、场景编辑等 —— 操作成功后扣费

## Assumptions (temporary, to validate)

1. **成功定义**：
   - 对于异步任务（story_analysis/image_generation/video_generation）：job 状态转移到 `succeeded` 时扣费
   - 对于同步操作（场景编辑）：HTTP 200 响应返回时扣费
   
2. **扣费失败处理**：
   - 若扣费失败（余额不足？系统故障？）：
     - 任务本身已完成，不应回滚
     - 应创建 `adjust` 记录标记异常，或进入待结算队列

3. **前端 UX**：
   - 操作前显示预期成本作为"估计值"（非保证金）
   - 操作后显示实际扣费

4. **支付充值流程**：
   - 暂不实现支付网关集成
   - 仅提供后端 API 供客户/管理员手工调用
   - 前端可选：简单的"联系销售充值"按钮

## Open Questions (high-value only)

**[Question 2] 具体操作类型的定价表是什么？**

需要确认：
- story_analysis: ? 积分/次
- image_generation: ? 积分/次
- video_generation: ? 积分/次
- 故事地图编辑、角色编辑、场景编辑各多少？
- 是否所有操作都扣费，还是某些操作免费？

**[Question 3] 扣费失败时的处理逻辑？**
- 若任务成功但扣费失败（如余额不足），是否：
  - A. 标记为 pending_billing，等待后续补扣？
  - B. 直接扣费失败，不记录任何交易？
  - C. 使用 adjust 记录标记异常，后台运营手工处理？

## Requirements (evolving)

* [ ] 定价规则模型（cost_catalog）—— 支持多种操作类型
* [ ] 扣费触发点集成（异步任务 success callback + 同步操作 HTTP handler）
* [ ] 扣费原子性保证（同一个操作只扣费一次）
* [ ] 扣费失败处理流程（可选：重试机制、待结算队列）
* [ ] 后端 API：`GET /api/v1/operation-costs` 返回定价表
* [ ] 后端 API：`POST /api/v1/wallet/preview-cost` 预览操作成本
* [ ] 前端 Wallet 页面：余额显示 + 流水列表
* [ ] 前端操作前弹窗：显示预期成本 + 确认扣费

## Acceptance Criteria (evolving)

* [ ] 后端：定价表完整定义，支持至少 5+ 个操作类型
* [ ] 后端：任务成功后自动创建 debit 交易
* [ ] 后端：同步操作完成后自动创建 debit 交易
* [ ] 后端：同一操作不重复扣费（幂等性）
* [ ] 后端：`GET /operation-costs` 返回定价表
* [ ] 后端：整合测试验证扣费流程（成功、失败、幂等等场景）
* [ ] 前端：Wallet 页面显示余额
* [ ] 前端：Wallet 页面显示最近 N 条流水（分页 or 滚动）
* [ ] 前端：操作前 toast/modal 显示预期成本
* [ ] 前端：操作后自动刷新余额和流水
* [ ] 文档：openapi.yaml 更新 operation-costs + preview-cost 端点
* [ ] 所有测试通过（`go test ./...` + `npm run build`）

## Definition of Done

* Tests added/updated (unit/integration for pricing logic)
* Lint / typecheck / CI green
* `go test ./...` 全绿
* `npm run lint && npm run build` (Studio) 全绿
* API docs (openapi.yaml) 更新
* 前端 Wallet 页面可交互（余额查询、流水查看）
* 后端扣费集成点完成（至少 story_analysis + image_generation）

## Out of Scope (explicit)

* [ ] 支付网关集成（Stripe / 支付宝 / 微信）——暂不做
* [ ] 发票生成 / 财务报表
* [ ] 动态定价 / 用户等级折扣
* [ ] 免费额度 / 试用期管理
* [ ] WebSocket 实时成本更新
* [ ] 成本预测 / ROI 计算
* [ ] 所有操作类型的扣费集成（MVP 先做核心 3 个：story_analysis / image_generation / video_generation）

## Technical Approach (evolving)

### 1. 后端架构

**Domain Layer**:
```go
// OperationCost 定价规则
type OperationCost struct {
    OperationType string
    Cost         int64  // 积分数
    Description  string
}

// 代码常量或从配置加载
var OperationCosts = map[string]int64{
    "story_analysis":     50,
    "image_generation":   100,
    "video_generation":   200,
    // ... 其他操作
}
```

**Service Layer**:
- `WalletService` 新增方法：
  - `GetOperationCost(operationType string) (int64, error)`
  - `Debit(ctx, operationType, refID) error` —— 扣费原子操作
- `StoryAnalysisService.OnAnalysisComplete()` —— 调用扣费
- `GenerationJobService.OnJobComplete()` —— 调用扣费

**HTTP Layer**:
- `GET /api/v1/operation-costs` —— 返回定价表
- `POST /api/v1/wallet/preview-cost` —— 预览成本
  - 入参：`{operation_type: "story_analysis"}`
  - 出参：`{operation_type, cost, description}`

**关键考量**：
- 扣费幂等性：用 `ref_id` (job_id / operation_id) 检查是否已扣过
- 事务保证：扣费和创建交易记录必须同步

### 2. 前端架构

**Pages**:
- `WalletPage.tsx` —— 余额显示、流水表格、充值按钮（可选）

**Hooks**:
- `useWallet()` —— GET /wallet，15s 轮询
- `useOperationCosts()` —— GET /operation-costs，缓存
- `usePreviewCost(operationType)` —— POST /wallet/preview-cost

**Components**:
- `WalletBalance` —— 显示余额 + 上次更新时间
- `TransactionList` —— 流水表格（支持分页 or 虚拟滚动）
- `CostConfirmDialog` —— 操作前确认弹窗
- 路由：`/wallet` 或集成到侧栏导航

### 3. 扣费触发点集成

**异步任务**（story_analysis / image_generation / video_generation）:
```go
// 在 job success 状态转移时调用
func (svc *WorkflowService) OnJobSucceeded(ctx, jobID) {
    // ... 业务逻辑
    
    // 扣费
    operationType := jobDetail.OperationType // "story_analysis", "image_generation", etc.
    _ = svc.walletService.Debit(ctx, operationType, jobID)
}
```

**同步操作**（场景编辑等）:
```go
// 在 HTTP handler 中
func (h *SceneHandler) UpdateScene(w http.ResponseWriter, r *http.Request) {
    // ... 验证 & 更新场景
    
    // 扣费
    _ = h.walletService.Debit(r.Context(), "scene_edit", sceneID)
    
    // 返回响应
    h.writeJSON(w, 200, sceneDTO)
}
```

## Decision (ADR-lite)

**Context**: 
- 需要决定计费时机（预付 vs 后付）
- 用户选择后付制（成功才扣费），降低用户操作门槛

**Decision**: 
- ✅ 采用**后付制 + 成功才扣费**
- ✅ 定价规则用**代码常量**（MVP 简单方案）
- ✅ 扣费触发点：异步任务 success + 同步操作 200 响应

**Consequences**: 
- 优点：用户不需提前检查余额，体验友好
- 缺点：若扣费失败需要特殊处理（待结算队列 or 运营介入）
- 风险：需要幂等性保证（avoid double-charge）

## Technical Notes

**Files & References**:
- `internal/domain/wallet.go`
- `internal/service/wallet_service.go`
- `internal/service/story_analysis_service.go` —— 集成扣费
- `internal/httpapi/generation_job.go` —— 集成扣费
- `apps/studio/src/api/hooks.ts`

**Constraints**:
- 组织隔离必须保持
- 交易审计完整
- 扣费幂等性（同一操作 + ID 只扣一次）

**Dependencies**:
- 需要了解 story_analysis / image_generation / video_generation 的 success callback 位置

---

## Next Steps (Updated)

**Immediate Q&A**:
1. ✅ [DECIDED] 计费时机 = 后付制（成功才扣费）
2. **[BLOCKING] 确认具体操作类型的定价表**
   - story_analysis: ? 积分
   - image_generation: ? 积分
   - video_generation: ? 积分
   - 其他操作的定价？
3. **[PREFERENCE] 扣费失败的处理逻辑**
   - A. pending_billing 队列（后续补扣）
   - B. 直接失败（不记录）
   - C. adjust 标记异常（运营手工处理）

After clarity on Questions 2&3 → Phase 1 实现（backend pricing model + debit endpoints）


---

## 定价方案最终确认

✅ **方案选择**：方案 A（均衡定价 - MVP）
✅ **Chat 计费**：后付制（成功才扣费）

### 最终定价表

| 操作类型 | 成本 | 计费时机 |
|---------|------|--------|
| Chat（对话） | 1 积分/次 | 成功返回后 |
| Story Analysis | 50 积分/次 | 任务 success 状态 |
| Image Generation | 100 积分/次 | 生成成功后 |
| Video Generation | 200 积分/次 | 生成成功后 |
| Storyboard Edit | 5 积分/次 | 编辑成功后 |
| Character Edit | 5 积分/次 | 编辑成功后 |
| Scene Edit | 5 积分/次 | 编辑成功后 |


---

## Question 3：扣费失败处理

✅ **决策**：A. Pending Billing Queue（待结算队列）

### 处理流程

1. 操作完成（成功）
2. 尝试扣费
3. 若扣费成功 → 创建 debit 交易，任务标记 settled
4. 若扣费失败 → 
   - 创建 pending_billing 记录
   - 标记任务为 awaiting_payment
   - Worker 定期重试（指数退避）
   - 若多次失败 → 标记异常，运营人员接管

---

## 最终实现计划

### Phase 1：后端定价模型 + Debit 核心能力

**Deliverables**:
- [ ] Domain: OperationCost / PendingBilling 类型
- [ ] Service: WalletService.Debit() 方法（幂等）
- [ ] HTTP: GET /operation-costs 端点
- [ ] HTTP: POST /wallet/preview-cost 端点
- [ ] Repo: pending_billing 表（持久化待结算）
- [ ] Repo: 幂等性检查（同 ref_id 只扣一次）
- [ ] 单元测试 + 集成测试（扣费成功/失败/幂等等）

**关键实现点**：
- 事务保证：扣费 + 交易记录必须原子
- 幂等性：用 (operation_type, ref_id) 检查是否已扣
- 错误处理：区分临时失败（重试）vs 永久失败（运营介入）

### Phase 2：异步任务集成 + Story Analysis 扣费

**Deliverables**:
- [ ] StoryAnalysisService.OnAnalysisComplete() 调用 Debit
- [ ] Image/Video GenerationService 类似
- [ ] Worker pending_billing 重试机制
- [ ] 集成测试：story_analysis 成功后自动扣费

**关键实现点**:
- 在 job success callback 中安全调用扣费
- 不影响主任务流程（错误 swallow）
- 重试策略（指数退避，最多 N 次）

### Phase 3：前端 Wallet 页面 + Chat 扣费

**Deliverables**:
- [ ] WalletPage.tsx 组件
- [ ] useWallet() 钩子（轮询）
- [ ] useOperationCosts() 钩子
- [ ] CostConfirmDialog 弹窗（操作前显示预期成本）
- [ ] TransactionList 表格（带分页）
- [ ] 集成到导航栏

**关键实现点**:
- 操作前显示预期成本确认
- 操作后自动刷新余额
- 美化设计（根据 ui-ux-pro-max）

### Phase 4：Chat 服务扣费集成

**Deliverables**:
- [ ] Chat API 返回成功后调用 Debit
- [ ] 前端显示对话成本
- [ ] 集成测试：chat 成功后扣费 1 积分

**关键实现点**:
- Chat 是轻量级操作，需要频繁交互，要确保不影响响应时间

---

## 实现路径（推荐顺序）

1. **PR1**: Backend pricing model + core Debit (Phase 1)
   - 目标：后端定价能力完整，可单独验证
   - 测试：单元 + 集成测试，go test 全绿
   
2. **PR2**: Story Analysis 扣费 + Pending Billing Worker (Phase 2)
   - 目标：第一个扣费集成点可工作
   - 测试：end-to-end 验证扣费流程
   
3. **PR3**: Frontend Wallet Page (Phase 3)
   - 目标：前端可查询、可显示
   - 测试：UI 交互可用
   
4. **PR4**: Chat 扣费集成 (Phase 4)
   - 目标：Chat 扣费完整
   - 测试：chat 后扣费成功

---

## Definition of Done (Updated)

* Tests: 每个 PR 都有相应测试覆盖
* Lint / Build: `go test ./...` + `npm run build` 全绿
* API Docs: openapi.yaml 更新所有新端点
* 前端集成：所有扣费页面可用
* Pending Billing: Worker 重试机制完善
* 文档：必要的设计文档、集成指南

---

## Out of Scope (Updated)

* Stripe/支付宝/微信 支付集成
* 财务报表 / 发票
* 动态定价 / 折扣 / 优惠券
* 成本预测 / 高级分析
* WebSocket 实时更新

