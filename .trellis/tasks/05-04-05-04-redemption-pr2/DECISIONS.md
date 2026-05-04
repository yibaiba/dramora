# PR2 最终决策记录

## 已确认的关键决策

### 1. 邮件分发模式
**决策**: 立即异步发送
- 调用 API 后立即投入队列
- 后台 worker 处理
- 用户立即看到反馈

### 2. QR 码在邮件中的呈现
**决策**: 内联 PNG（Base64 编码）
- 100% 邮件客户端兼容
- 用户直接扫码
- 符合行业标准（Amazon、Ticketmaster 等）

### 3. MVP 范围
**决策**: MVP+（包含模板定制）
- ✅ 列表输入赎回码
- ✅ 输入收件人邮箱列表 (逗号分隔)
- ✅ 自定义邮件主题
- ✅ 自定义邮件正文 (参数替换: {{code}}, {{shortUrl}}, {{expiresAt}})
- ✅ 邮件预览功能
- ✅ 异步后台处理

---

## 实现方案确定

| 功能 | 方案 | 依赖 |
|------|------|------|
| QR 码生成 | Go `go-qrcode` + Base64 | 标准库 |
| 邮件服务 | Resend API | RESEND_API_KEY env |
| 短链接 | Base62(code_id) | short_codes 表 |
| 模板引擎 | Go `text/template` | 标准库 |
| 后台队列 | 现有 inline_worker | 复用基础设施 |

---

## 预计工作量

- 后端: 8-10 小时
  - Base62 实现 + Base62 工具函数
  - short_codes 表 + 迁移
  - QR 码生成 (go-qrcode)
  - 邮件服务 (Resend API)
  - HTTP 端点 + 权限检查
  - 后台任务定义
  - 单元测试

- 前端: 4-5 小时
  - EmailDistributionDialog 组件
  - 模板编辑器 (参数替换)
  - API hooks (useEmailDistribution)
  - 邮件预览
  - 集成到 RedemptionCampaignManager

总计: 12-15 小时

---

## Out of Scope (PR3+)

- ❌ 收件人标签/分段
- ❌ A/B 测试
- ❌ 发送统计/打开率
- ❌ CSV 导入
- ❌ 定时发送
- ❌ 短信分发

