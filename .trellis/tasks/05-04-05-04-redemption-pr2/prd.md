# PR2: 赎回码系统 - 二维码 + 邮件分发

## 目标

在 PR1 基础上（核心生成/兑换/统计），实现完整的赎回码分发流程：二维码生成、邮件分发、短链接兑换。使管理员能够通过多种方式（二维码、邮件、短链接）分发赎回码。

## 我已知道的事实

**PR1 基础**:
- ✅ redemption_codes 表（乐观锁、status、expires_at）
- ✅ GenerateCodes、RedeemCode、GetCampaignStats API
- ✅ 权限检查（organization isolation）
- ✅ 钱包系统集成
- ✅ 前端兑换对话框 + 活动管理器

**项目约定**:
- API 路由: `/api/v1` 前缀
- 权限检查: 必须验证 organization_id
- POST 用于所有写操作（无 PUT/PATCH/DELETE）
- 前端使用 TanStack Query + Zustand
- 后端使用 Go 1.21 + Chi v5

## 临时假设（待验证）

1. QR 码应该返回图片 (PNG) 或可嵌入的 data URL？
2. 邮件应该集成第三方服务还是本地发送？
3. 短链接应该支持自定义或仅自动生成？
4. 邮件分发是同步还是异步（后台任务）？
5. 一个邮件活动可以分多批次发送吗？

## 开放问题（高价值）

- **技术选择**: QR 码库、邮件提供商、短链接算法
- **MVP 范围**: 哪些功能在 PR2 中，哪些延迟到 PR3+？
- **邮件分发模式**: 批量上传收件人还是单条测试？

## 需求（演进中）

### 功能需求 (MVP+)

1. **QR 码生成**
   - 内联 PNG 格式（Base64 编码）
   - 每个赎回码生成一个 QR 码
   - 大小: 200x200px (标准邮件尺寸)
   - 前端显示 + 邮件中包含

2. **邮件分发**
   - POST /api/v1/admin/redemption-codes:send-email
   - 输入: 
     - codes[] (赎回码列表)
     - recipients[] (邮箱列表，逗号分隔)
     - emailSubject (自定义主题)
     - emailBody (自定义正文，支持 {{code}}, {{shortUrl}}, {{expiresAt}}, {{amount}})
   - 异步处理（投入后台队列）
   - 返回: campaignId 用于跟踪

3. **短链接生成**
   - POST /api/v1/redemption-codes:shorten
   - 输入: code (赎回码)
   - 返回: shortCode, shortUrl
   - GET /r/:shortCode → 重定向到兑换页面

4. **邮件模板系统**
   - 预定义模板参数: {{code}}, {{shortUrl}}, {{expiresAt}}, {{amount}}, {{campaignName}}
   - 使用 Go text/template 进行参数替换
   - 前端预览功能 (模拟参数)

5. **前端集成**
   - EmailDistributionDialog 组件 (邮件分发对话框)
   - 模板编辑器 (主题 + 正文)
   - 邮件预览
   - 收件人列表输入
   - 集成到 RedemptionCampaignManager

### 技术需求

- Base62 编码实现 (ID ↔ shortCode 双向转换)
- short_codes 数据库表
- go-qrcode 库集成
- Resend API 集成 (RESEND_API_KEY 环境变量)
- 后台任务定义 (email_distribution 任务类型)
- 权限检查 (organization_id isolation)
- 邮件 HTML 模板

## 验收标准（演进中）

- [ ] QR 码生成和 Base64 编码正确
- [ ] Base62 short_code 生成无冲突
- [ ] 邮件包含内联 QR 码 (PNG Base64)
- [ ] 邮件包含短链接
- [ ] 短链接重定向到兑换页面正确
- [ ] 邮件模板参数替换正确 (code, shortUrl, expiresAt, 等)
- [ ] 邮件预览功能工作正确
- [ ] 邮件分发异步投入队列
- [ ] 所有操作进行权限检查（organization isolation）
- [ ] 短链接表数据完整
- [ ] 前端 EmailDistributionDialog 呈现正确
- [ ] 后端单元测试 > 80% coverage
- [ ] 代码质量: TypeScript/ESLint/Build 全通过
- [ ] 数据库迁移可正向/逆向执行

## 完成定义

- ✅ 后端单元测试（coverage > 80%）
- ✅ 前端 TypeScript 零错误
- ✅ ESLint 检查通过
- ✅ 代码格式 (gofmt) 正确
- ✅ 功能性集成测试（至少一个 e2e 路径）
- ✅ 文档/评论更新（如有新 API）
- ✅ 数据库迁移版本控制

## 范围外

- [ ] 短信分发（需要 SMS 提供商集成）
- [ ] 邮件模板的 WYSIWYG 编辑器（使用预定义模板）
- [ ] 邮件追踪像素/打开率统计
- [ ] 批量导入收件人的 CSV 解析
- [ ] A/B 测试不同邮件内容

## 技术方案（研究完成）

### 推荐方案

| 功能 | 技术选择 | 理由 |
|------|---------|------|
| **QR 码** | 前端生成 (qrcode.react) | 即时反馈、离线工作、用户可保存，后端无需处理图片 |
| **邮件** | Resend API | 现代、AI 友好、集成简单、自由套餐支持测试 |
| **短链接** | Base62(code_id) | 确定性、无冲突、简单快速、支持隐私（无需映射表） |
| **分发** | 异步后台任务 (worker 队列) | 复用现有 inline_worker 基础设施 |
| **模板** | Go text/template | 标准库，无外部依赖 |

### 数据库设计

**新表: short_codes**
```sql
CREATE TABLE short_codes (
  id SERIAL PRIMARY KEY,
  code VARCHAR(13) UNIQUE NOT NULL,       -- GIFT-XXXXXXXX
  short_code VARCHAR(8) UNIQUE NOT NULL,  -- base62(id)
  organization_id UUID NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  FOREIGN KEY(organization_id) REFERENCES organizations(id)
);
CREATE INDEX idx_short_codes_short_code ON short_codes(short_code);
```

**关键设计**:
- short_code 由 ID 的 Base62 编码生成（无需映射）
- 记录 organization_id 用于权限检查
- 轻量级表，仅用于正向查询

### 邮件服务集成

**环境变量**:
- RESEND_API_KEY=re_xxx (来自 https://resend.com)
- 本地环境可用 SendGrid 自由套餐替代

**邮件模板 (Go text/template)**:
```
Subject: 🎁 您的赎回码已准备好

Hi {{ .RecipientName }}!

获得积分: {{ .Amount }}

--- 方式 1: 直接兑换 ---
赎回码: {{ .Code }}

--- 方式 2: 点击链接 ---
{{ .ShortUrl }}

--- 方式 3: 扫码 ---
<二维码链接或内联>

有效期: {{ .ExpiresAt }}
```

### 文件引用

- 现有: `internal/repo/redemption_code_repo.go`
- 现有: `internal/service/redemption_code_service.go`
- 新增: `internal/repo/short_code_repo.go` (CRUD)
- 新增: `internal/service/email_service.go` (Resend 集成)
- 新增: `internal/httpapi/redemption_email.go` (POST 端点)
- 新增: `internal/httpapi/redemption_shortlink.go` (GET 重定向)
- 新增: DB 迁移 (short_codes 表)
- 前端: `apps/studio/src/studio/components/RedemptionQRDisplay.tsx`
- 前端: `apps/studio/src/studio/components/EmailDistributionDialog.tsx`

## 关键决策（待确认）

1. **邮件分发方式**: 立即发送 vs 定时发送？
2. **QR 码位置**: 邮件中显示 vs 作为附件/链接？
3. **短链接 URL**: 使用 /r/ 前缀 vs 其他？

## 已存档的决策

(待补充)
