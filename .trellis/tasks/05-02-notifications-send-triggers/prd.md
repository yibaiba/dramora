# 补完通知系统的发送触发点

## 目标

完成通知系统的最后一步：添加发送触发点，使钱包/邀请/计费操作能自动推送通知给用户。系统已 60% 完成，只需补充 3 个触发点即可全面启用。

## 现状

### ✅ 已完成
- 后端 DB：notifications 表已创建
- 后端 API：notification.go HTTP handler 完整（GET /notifications, PATCH mark-as-read）
- 前端 UI：NotificationsPage.tsx 已实现完整
- 前端 Hooks：useNotifications, useMarkNotificationAsRead, useMarkAllNotificationsAsRead

### ❌ 缺失
- wallet.go：充值/扣费后创建通知
- invitations.go：邀请操作后创建通知
- 后端发送 API：POST /api/v1/notifications（创建&发送）

## 需求

### 1. 钱包操作通知 (wallet_service.go)

当用户充值或扣费成功时，自动推送通知：

```
充值成功：
  标题："充值成功"
  内容："您已成功充值 1000 积分"
  操作：可点击跳转到 /wallet

扣费成功：
  标题："积分已扣除"
  内容："故事分析消耗 50 积分，剩余 950 积分"
  操作：可点击跳转到 /wallet/transactions
```

### 2. 邀请操作通知 (invitations.go)

当用户发送或接收邀请时，推送通知：

```
邀请发送：
  标题："邀请已发送"
  内容："您已邀请 user@example.com 加入"

邀请接收（广播到组织）：
  标题："新成员加入"
  内容："user@example.com 加入了你的组织"
  操作：可点击跳转到 /admin/members
```

### 3. 通知发送 API (notification.go)

补充后端的 POST /api/v1/notifications 端点：

```go
type CreateNotificationRequest struct {
  RecipientUserID *string  // 为 nil 时广播到组织全体
  OrganizationID  string
  Title           string
  Content         string
  ActionURL       string   // 可选：点击后跳转的 URL
  EventType       string   // "wallet_credit", "wallet_debit", "invitation_send", etc
  Metadata        map[string]interface{}
}

Response: Notification (201 Created)
```

## 验收标准

- [ ] wallet_service.go：Credit/Debit 后自动创建通知
- [ ] invitations.go：邀请操作后自动创建通知
- [ ] notification.go：POST /api/v1/notifications API 实现
- [ ] 前端 NotificationsPage 能显示新推送通知
- [ ] 通知中文显示正确
- [ ] 所有后端测试通过（go test）

## 工作量

- 后端代码修改：3-4 小时
- 前端测试验证：1 小时
- 总计：3-4 小时

## 实施步骤

1. **后端实现** (2-3h)
   - notification.go：添加 CreateNotification HTTP handler
   - wallet_service.go：在 Credit/Debit 处添加通知触发
   - invitations.go：在邀请操作处添加通知触发

2. **测试** (30min)
   - go test ./... 验证
   - 前端 NotificationsPage 刷新验证

3. **验证** (30min)
   - 完整流程测试（充值 → 扣费 → 通知显示）

## 完成定义

- ✅ 所有后端测试通过
- ✅ 前端能显示通知
- ✅ 中文文案正确
- ✅ git commit 记录清晰

