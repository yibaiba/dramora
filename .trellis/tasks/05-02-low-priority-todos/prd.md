# 3 个 LOW 优先级技术债 TODO

## 目标

完成 3 个积分/计费系统的优化 TODO，属于技术债类别，工作量约 1-2 周。这些都是后期功能增强，不是核心路径。

## 现状

- Phase 1-10 积分计费系统已完整交付 ✅
- 权限系统高风险漏洞已修复 ✅
- 3 个 MEDIUM TODO（日期过滤、系统接口）已完成 ✅

需要完成的 LOW 优先级 TODO：

1. **billing-report-export** - 导出报表为 CSV 格式
2. **billing-report-history-sidebar** - 历史报表侧栏快速查看
3. **advanced-pricing-ab-test** - 价格 AB 测试框架

## 我已知道的

### 现有 BillingReportsPage
- 前端已存在 `apps/studio/src/studio/pages/BillingReportsPage.tsx`
- 支持生成报表、分页、报表详情查看
- API hooks: useAdminBillingReports, useGenerateAdminBillingReport, useAdminBillingReportSummary

### 后端 billing_report 接口
- GET /api/v1/admin/billing-reports（列表）
- GET /api/v1/admin/billing-reports/{id}（详情）
- POST /api/v1/admin/billing-reports（生成）
- 数据库：billing_reports 表 + billing_breakdowns 表

### 前端交易历史页面
- `apps/studio/src/studio/pages/TransactionHistoryPage.tsx` 已存在
- 支持日期范围过滤、交易类型筛选

### 目前缺失的
- 报表导出功能（CSV）
- 历史报表快速查看侧栏
- 价格 AB 测试框架

## 临时假设

- CSV 导出不需要后端生成，前端直接导出表格数据
- 历史报表侧栏只需简单显示最近 N 个报表，支持切换
- AB 测试框架是后端配置 + 前端显示逻辑，不涉及真正的 A/B 分流

## 开放问题

1. **优先级顺序** - 三个 TODO 中哪个先做？建议顺序？
2. **CSV 导出策略** - 前端导出还是后端生成？格式如何定义？
3. **AB 测试框架** - 应该如何与现有的定价系统集成？

## 需求（已确认）

### TODO 1: billing-report-export（导出报表为 CSV）

**功能**：
- 在报表主表格中添加"导出"按钮（右上角）
- 点击后生成 CSV 文件并下载到本地
- CSV 包含列：周期开始、周期结束、操作类型、操作次数、总成本、平均成本

**实现**：
- 前端导出（浏览器直接生成 CSV）
- 无需后端支持

**文件修改**：
- `apps/studio/src/studio/pages/BillingReportsPage.tsx` - 添加导出按钮和逻辑
- 可选新建 `apps/studio/src/lib/csv-export.ts` - 导出工具函数

---

### TODO 2: billing-report-history-sidebar（历史报表侧栏）

**功能**：
- 在报表页面右侧添加只读侧栏，显示最近 10 个报表
- 点击侧栏中的报表项，主体切换显示该报表详情
- 侧栏显示信息：周期、状态、生成时间

**实现**：
- 前端新增侧栏组件，调用现有 API（useAdminBillingReports）
- 添加选中态和切换逻辑

**文件修改**：
- `apps/studio/src/studio/pages/BillingReportsPage.tsx` - 主页面布局改为 主体+侧栏
- 新建 `apps/studio/src/studio/components/ReportHistorySidebar.tsx` - 侧栏组件

---

### TODO 3: advanced-pricing-ab-test（价格 AB 测试框架）

**功能**：
- 后端支持 AB 测试配置（但初始状态为禁用）
- 前端根据用户/组织属于哪个 Variant，显示对应价格
- 支持激活/禁用 AB 测试的管理界面（admin 权限）

**实现**：
- 后端：新建 `ab_test_config.go` domain + `ab_test_service.go`
- 前端：修改价格计算逻辑，支持 AB test variant
- Admin 界面：添加 AB 测试管理页面

**用户分配策略**：
- 按组织 ID 哈希分配：`hash(organizationID) % 2`
- 保证同一组织始终看到同一测试方案

**文件修改**：
- 后端：domain, service, repo, http handler
- 前端：pricing.ts, OperationCostsAdminPage, 新建 ABTestManagementPage
- 数据库：无需新表（配置存储在 operation_costs_metadata 或单独配置表）

---

## 验收标准（已确认）

### TODO 1: export
- [ ] BillingReportsPage 中"导出"按钮可见
- [ ] 点击导出 → 生成 CSV 文件并下载
- [ ] CSV 文件格式正确，包含所有必要列
- [ ] 中文字符编码正确（UTF-8）
- [ ] 浏览器兼容（Chrome/Firefox/Safari）

### TODO 2: sidebar
- [ ] 右侧侧栏显示最近 10 个报表
- [ ] 点击侧栏项 → 主体切换显示该报表
- [ ] 侧栏显示：周期、状态、生成时间
- [ ] 样式与现有 UI 一致（Dark Mode 支持）
- [ ] 移动设备上隐藏侧栏（响应式）

### TODO 3: AB 测试
- [ ] 后端 AB 测试配置框架完成（初始禁用）
- [ ] 按组织 ID 哈希分配 variant
- [ ] 前端 pricing.ts 支持读取 AB test config
- [ ] 前端 OperationCostsTable 显示对应 variant 的价格
- [ ] Admin 界面支持查看和修改 AB 测试状态
- [ ] 单元测试覆盖：hash 分配、config 读写、前端渲染

---

## 完成定义

- 新功能单元测试 + 集成测试通过
- 前端构建无 lint 错误、TypeScript 通过
- 后端 go test ./... 通过（如涉及）
- 代码格式化（gofmt）
- 相关注释和文档更新
- 提交清晰的 git commit 信息

## 技术笔记

### 相关文件
- `apps/studio/src/studio/pages/BillingReportsPage.tsx` - 报表主页
- `internal/service/report_service.go` - 后端报表生成
- `internal/domain/billing_report.go` - 数据模型
- `apps/studio/src/api/types.ts` - 前端类型定义
- `apps/studio/src/api/hooks.ts` - React Query hooks

### 数据库
- `db/migrations/000025_create_billing_reports.up.sql` - 表结构
- billing_reports 表：id, organization_id, period_start, period_end, status, ...
- billing_breakdowns 表：report_id, operation_type, count, total_cost, ...

### 现有模式
- 前端导出：TransactionHistoryPage 可能有参考
- 分页 + 侧栏：OperationCostsAdminPage 可能有参考
- 功能开关：operation_costs_admin_page 使用了权限检查模式

