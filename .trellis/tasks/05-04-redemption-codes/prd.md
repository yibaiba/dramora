# 赎回码系统实现

## Goal

实现用户兑换赎回码（礼品卡/充值卡）获得积分（credits）的完整系统，包括：
- 后端：管理员生成批量赎回码，用户兑换赎回码获得积分
- 前端：用户输入赎回码界面，兑换确认，历史记录展示

## What I already know

从代码检查发现的现状：

### 后端基础设施 ✅
- ✅ 钱包系统已实现：`internal/service/wallet_service.go`
  - 组织级别钱包：balance (积分), updated_at
  - 交易日志：wallet_transactions 表，支持 kind/direction/reason/ref_type
- ✅ 交易记录完整：organization_id, actor_user_id, created_at, balance_after
- ✅ 积分模型已有：domain.OperationType（story_analysis, image_generation, video_generation 等）
- ✅ 钱包 API 已有：GET /api/v1/wallet (查询余额 + 最近 10 条流水)

### 前端钱包 UI ✅
- ✅ WalletPage 已存在
- ✅ WalletBalanceCard 显示余额
- ✅ ChargeWalletDialog 处理充值流程
- ✅ TransactionHistoryPage 展示交易历史

### 数据库
- ✅ wallets 表：organization_id, balance, updated_at
- ✅ wallet_transactions 表：id, organization_id, kind, direction, amount, reason, ref_type, ref_id, balance_after, actor_user_id, created_at

## Assumptions (temporary - to validate)

1. **赎回码格式**：随机字符串（如：GIFTCARD-XXXX-XXXX-XXXX）还是数字？
2. **赎回权限**：是否只有 admin 可以生成批量赎回码？普通用户能否查看已兑换历史？
3. **有效期**：赎回码有没有过期时间限制？一个码只能用一次还是多次？
4. **金额灵活性**：每个赎回码固定金额（如 100 积分）还是可配置金额？
5. **兑换流程**：用户是否需要输入兑换码？是否支持批量兑换？

## Open Questions (blocking/preference)

### 1. 赎回码应该是什么格式？
建议：随机字符串（8-12 字符），易于手动输入和验证

### 2. 支持哪些赎回场景？
- A. 仅通过管理后台生成，用户输入码兑换
- B. 支持二维码扫描兑换
- C. 邮件/短链接自动兑换（需要超链接）
- 建议：先实现 A，为 B/C 预留扩展点

### 3. 赎回码的有效期和使用限制？
- 期限：永久有效、30 天有效、或按生成时间设定？
- 使用限制：每个码仅一次、限定用户、还是任意用户?
- 建议：支持按生成时间设定有效期，每个码最多一次使用

## Requirements (evolving)

### 选定方向：高级版 (方向 C)
支持二维码、邮件分发、短链接自动兑换 + 营销活动关联 + 实时仪表板

### 后端功能

#### 1. 数据库表
- [ ] redemption_codes：赎回码主表（id, code, amount, campaign_id, status, created_by, created_at, used_by, used_at, expires_at, reason）
- [ ] redemption_campaigns：活动表（id, name, description, status, created_by, created_at, redemption_count, total_amount）
- [ ] redemption_distribution：分发记录表（id, code, distribution_method, recipient_email, sent_at, valid_until）

#### 2. 核心 API
- [ ] POST /api/v1/admin/redemption-campaigns:create - 创建营销活动
- [ ] POST /api/v1/admin/redemption-codes:generate - 批量生成赎回码（与活动关联）
- [ ] POST /api/v1/admin/redemption-codes:export - 导出赎回码为 CSV
- [ ] GET /api/v1/admin/redemption-campaigns/{campaignId}/dashboard - 活动进度仪表板
- [ ] POST /api/v1/redemption-codes:redeem - 用户通过码兑换（支持输入框）
- [ ] GET /api/v1/redemption-codes/verify/{code} - 验证码有效性（用于短链接）
- [ ] POST /api/v1/redemption-codes:redeem-via-link - 通过短链接/邮件链接一键兑换

#### 3. 二维码支持
- [ ] 生成二维码（GET /qr?code=GIFT-XXX）
- [ ] 前端展示可下载的二维码

#### 4. 邮件分发
- [ ] 生成兑换链接（如 https://dramora.com/redeem?token=xxxx）
- [ ] 邮件模板（HTML）包含链接、二维码、金额说明
- [ ] 邮件发送队列（异步）
- [ ] 发送记录跟踪

#### 5. 验证与安全
- [ ] 赎回码有效性检查（未使用、未过期、不超额）
- [ ] 原子性操作：检查+标记+交易一体化，防止并发冲突
- [ ] Token 过期保护（邮件链接有效期 7 天）

### 前端功能

#### 1. 管理后台
- [ ] 活动管理页面：创建活动、设置名称、有效期、赎回码数量
- [ ] 赎回码生成界面：输入数量、单个金额、导出为 CSV/二维码
- [ ] 活动仪表板：
  - 实时显示已兑换数/总数（进度条）
  - 已生成金额 vs 已兑现金额统计
  - 时间趋势图（日兑换数）
  - 导出兑换明细

#### 2. 用户兑换界面
- [ ] 钱包页面新增"兑换赎回码"标签页
- [ ] 手动输入框：输入码、大小写不敏感、自动去空格
- [ ] 邮件自动兑换：点击邮件链接自动填充码并确认
- [ ] 二维码扫描：支持手机扫描二维码自动跳转兑换
- [ ] 确认对话框：显示码、金额、当前余额、兑换后余额
- [ ] 成功反馈：Toast 通知、余额实时更新

#### 3. 兑换历史
- [ ] TransactionHistoryPage 支持筛选"赎回码"交易
- [ ] 展示活动名称、兑换码、兑换时间、金额

## Acceptance Criteria (evolving)

- [ ] 后端能生成唯一的赎回码（不重复）
- [ ] 用户能正确兑换有效的赎回码并获得积分
- [ ] 重复兑换同一赎回码被拒绝
- [ ] 过期赎回码无法兑换
- [ ] 交易历史正确记录赎回操作
- [ ] 前端显示清晰的成功/失败提示
- [ ] 所有 TypeScript/ESLint/Go 测试通过

## Definition of Done

- [ ] 代码：Go 后端 + TypeScript 前端完整实现
- [ ] 测试：单元测试覆盖主要路径（生成、兑换、验证、重复、过期）
- [ ] 集成测试：通过 API 端到端验证
- [ ] 质量：TypeScript/ESLint 通过、Go tests 通过、构建成功
- [ ] 文档：更新 README 或 API 文档说明赎回码功能
- [ ] 部署：本地验证通过

## Out of Scope (explicit)

- [ ] ~~支持赎回码的二级代理销售（经销商管理）~~ - 后续考虑
- [ ] ~~批量导入赎回码 CSV（直接上传已有码）~~ - 后续考虑
- [ ] ~~基于地理/组织维度的细粒度权限控制~~ - 后续考虑
- [ ] ~~第三方分发渠道集成（如社交媒体分享）~~ - 后续考虑

## Implementation Plan

**PR1（核心 + 活动管理）**：预计 2-4 天
- 后端：赎回码表、活动表、生成 API、兑换 API、验证逻辑、统计 API、权限检查
- 前端：输入框、手动兑换流程、活动创建页面、仪表板展示
- 测试：单元测试、集成测试、API 端到端测试
- 交付验收：用户能通过输入码兑换积分、管理员能创建活动和查看统计

**PR2（二维码 + 邮件分发）**：后续 PR，预计 2-3 天
- 后端：QR 生成 API、邮件发送队列、短链接兑换、分发记录表
- 前端：二维码展示、邮件一键兑换、分发流程
- 交付验收：完整的分发渠道支持

## Technical Approach

### 数据库设计

新建表 `redemption_codes`:

```sql
CREATE TABLE IF NOT EXISTS redemption_codes (
    id TEXT PRIMARY KEY,
    code TEXT UNIQUE NOT NULL,  -- 赎回码，如 "GIFT-XXXXX"
    organization_id TEXT NOT NULL,  -- 拥有者组织
    amount BIGINT NOT NULL,  -- 积分金额
    status TEXT NOT NULL DEFAULT 'unused',  -- unused / used / expired
    created_by TEXT NOT NULL,  -- 创建者 admin user_id
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used_by TEXT,  -- 兑换者 user_id
    used_at TIMESTAMPTZ,  -- 兑换时间
    expires_at TIMESTAMPTZ,  -- 过期时间（null = 永不过期）
    reason TEXT  -- 备注，如 "月度奖励" 或 "合作方赠送"
);

CREATE INDEX IF NOT EXISTS redemption_codes_org_status_idx 
    ON redemption_codes (organization_id, status);
CREATE INDEX IF NOT EXISTS redemption_codes_code_idx 
    ON redemption_codes (code);
```

### 后端 API

**生成赎回码** (Admin 权限):

```
POST /api/v1/admin/redemption-codes:generate
Content-Type: application/json

{
  "count": 100,  -- 生成数量
  "amount": 1000,  -- 每个码的积分
  "expires_at": "2025-06-04T00:00:00Z",  -- 可选，过期时间
  "reason": "月度奖励"  -- 可选，备注
}

Response (200):
{
  "codes": [
    "GIFT-ABC123",
    "GIFT-DEF456",
    ...
  ],
  "count": 100
}
```

**用户兑换赎回码**:

```
POST /api/v1/redemption-codes:redeem
Content-Type: application/json

{
  "code": "GIFT-ABC123"
}

Response (200):
{
  "success": true,
  "amount": 1000,
  "new_balance": 5000,
  "message": "恭喜！你获得了 1000 积分"
}

Error (422):
{
  "error": "code_already_used",  -- 或 "code_not_found", "code_expired"
  "message": "赎回码已被使用"
}
```

### 前端实现

1. **输入界面**：在 WalletPage 新增"兑换赎回码"标签页或对话框
2. **输入字段**：文本输入框，大写自动转换，自动去空格
3. **兑换按钮**：提交前验证格式
4. **反馈**：成功提示（Toast/Modal）显示新余额，失败显示错误信息
5. **历史记录**：TransactionHistoryPage 中 kind='redemption_code' 的交易

## Technical Notes

### 文件清单（待创建/修改）

后端:
- `db/migrations/000XXX_create_redemption_codes.up.sql` - 新建表
- `internal/domain/redemption_code.go` - 领域模型
- `internal/repo/redemption_code_repo.go` - 数据库操作
- `internal/service/redemption_code_service.go` - 业务逻辑
- `internal/httpapi/redemption_code.go` - HTTP handlers
- `internal/httpapi/redemption_code_test.go` - 测试

前端:
- `apps/studio/src/studio/components/RedemptionCodeDialog.tsx` - 输入/兑换对话框
- `apps/studio/src/api/hooks.ts` - 新增 useRedeemCode hook
- `apps/studio/src/studio/pages/WalletPage.tsx` - 集成对话框

### 相关文件
- 钱包服务：`internal/service/wallet_service.go`
- 交易处理：`internal/repo/wallet_repo.go`
- 钱包前端：`apps/studio/src/studio/pages/WalletPage.tsx`

### 实现原则
- 遵循现有钱包事务处理逻辑（direction=1 表示进账）
### 并发控制策略：乐观锁 + 重试（方案 C）

使用 `version` 字段追踪状态变化，避免死锁风险，支持高并发：

```sql
CREATE TABLE IF NOT EXISTS redemption_codes (
    id TEXT PRIMARY KEY,
    code TEXT UNIQUE NOT NULL,
    organization_id TEXT NOT NULL,
    campaign_id TEXT,
    amount BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'unused',  -- unused / used
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used_by TEXT,
    used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    reason TEXT,
    version INT NOT NULL DEFAULT 0  -- 乐观锁字段
);

-- 兑换时的原子操作
UPDATE redemption_codes 
SET used_by=$1, used_at=NOW(), status='used', version=version+1
WHERE code=$2 AND status='unused' AND version=$3;
-- 检查 affected_rows：
--   > 0: 成功兑换
--   0: 状态冲突（已被兑换或已过期），需要重试或返回友好错误
```

优点：
- 高并发环境无死锁、性能优异
- 业界标准做法（乐观锁适合读多写少场景）
- 前端可缓存版本信息，减少重试

实现细节：
- 后端重试逻辑：最多重试 3 次，每次间隔 100ms
- 前端错误处理：用户看到"码已被使用"或"系统繁忙，请重试"

