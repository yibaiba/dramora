# WorkspacePage 团队空间管理 - 需求规划文档

## 🎯 Goal

为 dramora 项目实现 **WorkspacePage** 组件，支持团队空间（workspace）的管理和浏览。工作空间是组织下的逻辑分组单位，用于隔离不同的项目或团队协作环境。

## 📌 What I Already Know

### 现有上下文
1. **应用架构**
   - 前端路由系统在 `apps/studio/src/studio/routes.ts` 定义
   - 所有页面都在 `apps/studio/src/studio/pages/` 目录
   - 导航菜单项在 `studioNavItems` 数组中
   - 当前 WorkspacePage **不存在**（未在路由或导航中定义）

2. **后端 API 线索**
   - 存在 `/episodes/{episodeId}/storyboard-workspace` API 端点
   - `internal/service/storyboard_workspace_service.go` 存在后端服务

3. **相关页面参考**
   - InvitationsPage (38KB) - 团队邀请管理
   - AdminSettingsPage (38KB) - 系统设置管理
   - HomePage (20KB) - 项目总览
   - GalleryPage (11KB) - 素材库

4. **技术栈**
   - React + TypeScript + React Router
   - React Query for data fetching
   - Tailwind CSS for styling
   - TanStack Table for complex data tables
   - Lucide React for icons

### 已确认的设计约定
- 所有页面使用一致的导航、加载、错误状态处理
- 表格页面使用 TanStack Table v8（见 BatchQueueTable 示例）
- 所有 API 调用通过 React Query hooks
- 列表页面通常支持分页、排序、筛选
- 编辑操作多使用模态框或独立编辑页面

## ❓ Open Questions (需要用户澄清)

1. **WorkspacePage 的主要功能是什么？**
   - 是否只显示/浏览工作空间列表？
   - 是否支持创建/编辑/删除工作空间？
   - 是否需要成员管理？

2. **工作空间的业务模型**
   - 工作空间属于组织还是项目？
   - 工作空间有哪些关键属性（名称、描述、权限、成员等）？
   - 用户访问权限如何控制？

3. **UI/UX 设计**
   - 应该是表格列表还是卡片网格？
   - 是否需要搜索、筛选、排序功能？
   - 编辑工作空间用模态框还是单独页面？

4. **后端 API 状态**
   - 是否已有完整的 workspace CRUD API？
   - API 响应格式是什么？
   - 是否有权限校验需求？

## 📋 Requirements (待更新)

- [ ] 确定功能范围
- [ ] 确定 UI 设计方向
- [ ] 确定后端 API
- [ ] 确定数据结构

## ✅ Acceptance Criteria (待更新)

- [ ] WorkspacePage 组件实现
- [ ] 页面在路由中正确注册
- [ ] 导航菜单显示 WorkspacePage 项
- [ ] TypeScript 编译无错误
- [ ] ESLint 通过
- [ ] 前端构建成功

## 📝 Definition of Done (团队质量标准)

- [ ] 代码编译无 TypeScript 错误
- [ ] ESLint 无新警告/错误
- [ ] 所有 React Query hooks 正确使用
- [ ] 加载/错误/空状态都正确处理
- [ ] 可访问性：ARIA 标签、语义 HTML
- [ ] 响应式设计（桌面/平板/手机）
- [ ] 如需 API 调用，类型安全且异常处理完整

## 🚫 Out of Scope (初期不包含)

- WebSocket 实时更新
- 高级工作空间模板系统
- 工作空间内部的详细配置编辑
- 工作空间的性能分析和使用统计
- 工作空间级别的权限模型细化（涉及大量后端配置）

## 🔗 Technical Notes

### 文件位置参考
- 页面：`apps/studio/src/studio/pages/WorkspacePage.tsx`
- API hooks：`apps/studio/src/api/hooks.ts` (添加新的 workspace hooks)
- API types：`apps/studio/src/api/types.ts` (添加 Workspace 类型定义)
- API client：`apps/studio/src/api/client.ts` (添加 workspace API 调用)
- 路由：`apps/studio/src/studio/routes.ts` (添加导航项)
- 主 App：`apps/studio/src/App.tsx` (添加路由)

### 技术约束
- 必须使用 React 18 + TypeScript 严格模式
- 所有 props 必须完全类型化（不允许 `any`）
- 必须使用 React Query for data fetching（不直接使用 fetch）
- 组件长度 ≤ 50 行（超过则拆分）
- 必须支持深色模式（Tailwind 自动处理）

### 参考实现
- **类似表格页面**: BatchQueueTable (8,550 行 - 功能完整)
- **类似管理页面**: InvitationsPage (35KB - 创建/编辑/删除)
- **类似设置页面**: AdminSettingsPage (38KB - 复杂表单)

---

## 📌 Next Steps

1. 用户确认 WorkspacePage 的功能范围和业务模型
2. 设计 UI 原型或参考现有页面
3. 确认后端 API 可用性
4. 实现组件并验证


---

## 🎯 Requirement Gathering Progress

### ✅ Question 1: Functionality Scope
**Answer**: 仅显示/浏览 (view_only)
**Decision**: WorkspacePage 将是一个**只读列表页面**，用户可以浏览团队工作空间，但不支持创建、编辑或删除操作。

**Implications**:
- UI: 简单的表格或卡片列表（无编辑按钮）
- 功能: GET 工作空间列表 API
- 用户操作: 搜索、筛选、排序、可能的跳转到工作空间详情
- Scope: 减小（不需要编辑表单、模态框等）


### ✅ Question 2: Data Display Scope
**Answer**: 标准 (standard)
**Decision**: WorkspacePage 表格将显示 5 列：名称、描述、成员数、项目数、创建时间

**Table Columns**:
1. **Workspace Name** (必需) - 工作空间名称，主要标识
2. **Description** - 工作空间描述/说明
3. **Members** - 成员数量 (数字 or avatar group)
4. **Projects** - 该工作空间下的项目数
5. **Created At** - 创建时间 (格式化时间戳)

**Optional Interactions**:
- 点击行跳转到工作空间详情页面（如果有）
- 表头排序（如按名称、创建时间等）
- 简单搜索/筛选（按名称）
- 分页（如果工作空间很多）


### ✅ Question 3: UI Design Style
**Answer**: 卡片网格 (cards)
**Decision**: WorkspacePage 将使用**响应式卡片网格**展示工作空间

**Card Layout**:
- Grid: 响应式布局 (md:grid-cols-2 lg:grid-cols-3)
- Card 内容:
  - 工作空间名称 (标题)
  - 描述 (副标题/描述)
  - 统计: 成员数 | 项目数
  - 创建时间 (底部灰色文字)
- 交互: 
  - Hover 效果 (背景色变化、投影)
  - 可点击跳转到工作空间详情（如果路由存在）
- 加载状态: Skeleton loading cards
- 空状态: "暂无工作空间"

---

## 📋 Final Requirements (已收敛)

### 功能需求
- [x] 只读浏览页面（无创建、编辑、删除）
- [x] 工作空间列表查询 API
- [x] 显示 5 列信息（名称、描述、成员数、项目数、创建时间）
- [x] 响应式卡片网格 UI
- [x] 加载状态处理
- [x] 空状态处理
- [x] 错误状态处理

### 技术需求
- [x] React 18 + TypeScript
- [x] React Router 集成
- [x] React Query 数据获取
- [x] Tailwind CSS 样式
- [x] 完整类型定义（无 any）
- [x] 暗黑模式支持

### 非功能需求
- [x] 可访问性 (ARIA 标签、语义 HTML)
- [x] 响应式设计 (移动/平板/桌面)
- [x] 遵循应用设计规范

---

## ✅ Acceptance Criteria

- [ ] WorkspacePage.tsx 组件创建和实现
- [ ] 卡片网格UI正确显示工作空间信息
- [ ] 工作空间列表 API 集成 (通过 React Query)
- [ ] 加载/错误/空状态都有正确的 UI 反馈
- [ ] TypeScript 编译无错误
- [ ] ESLint 通过检查
- [ ] 前端构建成功 (npm run build)
- [ ] 后端测试通过 (go test ./...)
- [ ] 页面在路由中正确注册
- [ ] 导航菜单显示 WorkspacePage 链接
- [ ] 响应式设计在不同设备上工作正常

---

## 🎨 Implementation Plan

### PR1: 基础脚手架 + API 层
- 创建 WorkspacePage.tsx 组件（空实现）
- 在 routes.ts 添加导航项和路由路径
- 在 App.tsx 注册路由
- 在 types.ts 定义 Workspace 类型
- 在 client.ts 创建 listWorkspaces API 函数
- 在 hooks.ts 创建 useWorkspaces React Query hook

### PR2: UI 实现 + 集成
- 实现卡片网格 UI (Tailwind CSS)
- 集成 useWorkspaces hook
- 处理加载/错误/空状态
- 添加响应式设计
- 确保暗黑模式支持

### PR3: 质量验证 + 优化
- 运行 TypeScript 编译检查
- 运行 ESLint 检查
- 前端构建验证
- 后端测试验证
- 最终样式调整和文档

---

## 🔗 Technical Notes

### API Requirements
需要后端提供:
- `GET /api/v1/workspaces` - 获取工作空间列表
- 响应格式: 
  ```json
  {
    "data": [
      {
        "id": "workspace_001",
        "name": "Marketing Team",
        "description": "Marketing campaigns and assets",
        "members_count": 5,
        "projects_count": 3,
        "created_at": "2024-05-01T10:00:00Z"
      }
    ]
  }
  ```

### Type Definitions (types.ts)
```typescript
type Workspace = {
  id: string
  name: string
  description?: string
  members_count: number
  projects_count: number
  created_at: string
}

type ListWorkspacesResponse = {
  data: Workspace[]
}
```

---

**Status**: ✅ 需求梳理完成，可以开始实现

