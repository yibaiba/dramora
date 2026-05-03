# GalleryPage 高级功能 - 批量操作、资产对比、导出

## Goal

为 Dramora Studio Gallery 添加高级功能，支持用户批量管理资产、对比多个版本、导出素材包，提升资产管理效率和专业度。

## What I already know

* GalleryPage MVP 已完成（3 列网格、筛选、统计卡片）
* 当前支持：按类型、按状态筛选
* 现有代码在 `apps/studio/src/studio/pages/GalleryPage.tsx`
* 资产数据结构：`{ id, purpose, uri, status, metadata }`
* 使用 React Query + Zustand 状态管理
* Tailwind v4 + 自定义 CSS

## Assumptions (temporary)

* 批量操作包括：删除、标记为最佳版本、锁定、导出
* 资产对比器显示 2-3 个版本的并排对比
* 导出素材包为 ZIP 格式，包含选中资产
* 权限检查：只能操作自己的资产
* 所有操作需要确认对话框

## Open Questions

* 批量操作的具体范围（删除/标记/锁定/导出）？
* 资产对比器应该对比哪些内容？
* 导出包的命名规则和内容结构？

## Requirements (evolving)

* ✅ 批量操作范围：基础版（删除 + 导出）
* [ ] 批量选择（多选复选框）
* [ ] 删除操作（含确认对话框）
* [ ] 导出素材包（ZIP 格式）
* [ ] 批量操作工具栏（动态显示）

## Acceptance Criteria (evolving)

* [ ] 用户可勾选多个资产
* [ ] 批量删除成功，数据库同步
* [ ] 可对比 2-3 个资产版本
* [ ] 导出 ZIP 包含所选资产
* [ ] 所有操作有确认提示
* [ ] TypeScript 零错误
* [ ] ESLint 通过
* [ ] 前端构建成功

## Definition of Done

* Tests added/updated (unit/integration)
* Lint / typecheck / CI green
* Docs/notes updated if behavior changes
* 手动验证：批量操作、对比、导出流程

## Out of Scope (explicit)

* 资产对比器（保留到下个周期）
* 实时协作编辑
* 资产版本历史记录
* 智能去重
* 高级排序
* 收藏夹管理
* 撤销/重做功能

## Technical Notes

* **GalleryPage 位置**: `apps/studio/src/studio/pages/GalleryPage.tsx`
* **相关 Hook**: useEpisodeAssets(), useAssetMutations()
* **API 约定**: GET /api/v1/episodes/{id}/assets, POST assets:delete, etc.
* **权限**: 需要检查 organizationId 匹配
* **样式**: Tailwind v4, 深色主题

## Research Notes (to be filled)

### 研究方向
* 批量操作的最佳实践（Figma、Notion 等）
* ZIP 导出库选择（jszip？）
* 对比器 UI 设计（Tailwind 实现）

## Decision (ADR-lite)

**Context**: GalleryPage MVP 已完成，需要添加高级功能以提升用户体验。

**Decision**: 本周期专注"批量操作 MVP"（删除 + 导出），对比器保留到下周期。

**Consequences**: 
- ✅ 交付速度快（2-3 天）
- ✅ 简化代码复杂度
- ✅ 保留对比器作为 Phase 2
- ⚠️ 用户暂时无法对比版本

---

## Technical Approach

### 1. 数据模型
- `selectedAssetIds: Set<string>` - 批量选择状态
- 本地状态（Zustand）管理

### 2. UI 组件
- **AssetCard 增强** - 添加复选框
- **SelectionToolbar** - 批量操作工具栏（显示选中数量、操作按钮）
- **ConfirmDialog** - 删除确认对话框
- **ExportProgressDialog** - 导出进度条

### 3. 操作流程
```
用户点击资产卡片的复选框
  ↓
更新 selectedAssetIds
  ↓
显示 SelectionToolbar（删除、导出按钮）
  ↓
用户点击删除/导出
  ↓
显示确认对话框
  ↓
API 调用：DELETE /api/v1/assets/{id} 或 POST /api/v1/assets:export
  ↓
成功后更新 UI
```

### 4. 导出实现
- 使用 jszip 库生成 ZIP 包
- ZIP 命名：`assets_${episodeId}_${timestamp}.zip`
- 包含结构：`/assets/{kind}/{id}.{ext}`

### 5. 权限检查
- 前端：检查 asset.organizationId === currentOrganization
- 后端：再次验证权限

---

## Implementation Plan (small PRs)

### PR1: 批量选择 UI (1 天)
- AssetCard 复选框
- 批量选择逻辑
- SelectionToolbar 基础框架
- 样式 CSS

### PR2: 批量删除 (1 天)
- 删除 API 集成
- ConfirmDialog 实现
- 错误处理
- 成功反馈

### PR3: 导出功能 (1 天)
- jszip 集成
- ExportProgressDialog 实现
- ZIP 生成逻辑
- 浏览器下载

**总计**: 3 天 ≈ 1 个工作周

---

## Acceptance Criteria (updated)

- [ ] 用户可通过复选框多选资产
- [ ] 删除批量资产，数据库同步
- [ ] 导出 ZIP 包含所选资产
- [ ] 所有操作有确认提示
- [ ] 批量选择数量显示正确
- [ ] TypeScript 零错误，ESLint 通过
- [ ] 前端构建成功 (< 850KB gzip)
- [ ] 手动验证：删除、导出流程

