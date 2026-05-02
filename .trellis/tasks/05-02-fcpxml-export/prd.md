# FCPXML 导出 - 时间线导出为编辑软件格式

## 目标

实现时间线导出为 FCPXML 格式，使用户能在 Final Cut Pro、DaVinci Resolve 等专业编辑软件中打开和编辑生成的视频时间线。

## 现状

### ✅ 已有基础
- 后端 Timeline 数据模型完整：Timeline, TimelineTrack, TimelineClip
- 后端 API：GET /episodes/{episodeId}/timeline 获取时间线数据
- 前端 TimelineExportPage.tsx 已实现基础框架
- 后端 Export 表和相关接口存在

### ❌ 缺失
- FCPXML 生成逻辑（前端）
- FCPXML 导出按钮和交互（前端）
- 无后端 FCPXML 生成所需

## 需求

### 功能
1. 在 TimelineExportPage 添加"导出 FCPXML"按钮
2. 用户点击后，前端获取时间线数据 → 生成 FCPXML 文件 → 下载到本地
3. FCPXML 文件格式符合 FCP 规范，支持在 Final Cut Pro 打开

### 技术选型
- 使用 `xml` npm 包生成 XML
- 纯前端实现，无需后端修改

### 数据映射
```
Timeline    →  FCPXML
├─ tracks   →  <sequence><spine>
├─ clips    →  <clip> with timecode/duration
├─ assets   →  <media><video> with path/URL
└─ duration →  <duration> in frames
```

### MVP 范围
- ✓ 支持视频轨道（V1）
- ✗ 不支持音频轨道（后期补充）
- ✗ 不支持转场/特效（后期补充）
- ✗ 不支持字幕（后期补充）

## 验收标准

- [ ] TimelineExportPage 中"导出 FCPXML"按钮可见
- [ ] 点击导出 → 生成 FCPXML 文件
- [ ] 文件可在 Final Cut Pro 打开
- [ ] 时间码、镜头顺序、时长正确映射
- [ ] 前端无 TypeScript 错误
- [ ] 前端无 lint 错误
- [ ] 文件名格式：episode-{episodeId}-timeline-{timestamp}.fcpxml

## 工作量

- 前端代码实现：2-3 小时
- 测试和验证：1 小时
- 总计：3-4 小时

## 实施步骤

1. **创建 FCPXML 生成工具** (utils/fcpxml-generator.ts)
   - Timeline → FCPXML 转换逻辑
   - 时间转换（ms → frames @ 30fps）

2. **修改 TimelineExportPage** (TimelineExportPage.tsx)
   - 添加导出按钮
   - 绑定点击事件
   - 调用生成和下载

3. **前端验证**
   - npm run build
   - 无 lint/TypeScript 错误

4. **手动测试**
   - 导出 FCPXML
   - 在 Final Cut Pro 打开验证

## 完成定义

- ✅ 前端构建无错误
- ✅ TypeScript 通过
- ✅ 无 lint 警告
- ✅ FCPXML 文件可生成
- ✅ 文件可在编辑软件打开

