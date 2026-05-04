# 📱 电商 AI 短视频生成 - 需求文档 (MVP v1)

## 🎯 核心目标

为 Dramora Studio 提供**一键 AI 电商短视频生成**功能，让电商卖家无需任何视频编辑技能，选择模板 + 填商品信息 → 自动生成 → 15-30 分钟内获得可直接发布的短视频。

**目标用户**: 电商卖家（0-3 年经营经验，无视频技能）
**核心价值**: 一键生成 + 自动化 + 傻瓜操作
**市场空白**: 市面上没有完整的电商 AI 短视频一键生成产品

---

## 📊 与现有系统的关系

```
现有工作流:
  GalleryPage (视频库) 
    ↓
  StoryboardPage (AI 分镜) 
    ↓
  EditVideoPage (多轨编辑 - 给专业编导用)
    ↓
  [NEW] ShortVideoPage (AI 模板 - 给电商卖家用) ← 本次项目
    ↓
  QueuePage (批量生成队列)
```

**核心差异**:
- EditVideoPage: 手工编辑，高复杂度
- ShortVideoPage: **自动生成**，傻瓜操作

---

## ✨ MVP 范围（优化版 5.5 周）

### 模板（初期 2-3 个，后续扩展到 5 个）

1. **商品聚焦** (Product Showcase - 虚拟主播)
   - 构成: 虚拟主播展示 + 商品细节 + 购买 CTA
   - 时长: 15s
   - 参数: 商品名、价格、SKU
   - AI: HeyGen 虚拟主播 + Claude 文案

2. **活动推广** (Campaign Promotion - 虚拟主播)
   - 构成: 虚拟主播讲解活动 + 商品展示 + 折扣信息
   - 时长: 15s
   - 参数: 活动标题、原价、折扣价
   - AI: HeyGen 虚拟主播 + Claude 文案

3. **使用场景** (Use Case - 虚拟主播)
   - 构成: 虚拟主播演示使用场景 + 效果展示
   - 时长: 20s
   - 参数: 场景描述、产品优势
   - AI: HeyGen 虚拟主播 + Claude 文案

### 技术栈

**后端**:
- ShortVideoTemplate 数据模型
- ShortVideo 数据模型
- HeyGen API 集成
- Claude API 集成
- Job 编排和依赖管理

**前端**:
- ShortVideoPage 主页面
- TemplateSelector (侧边栏)
- ParametersPanel (参数编辑)
- PreviewPanel (实时预览/进度)
- ProgressTracker UI

**集成**:
- Timeline 生成引擎 (参数 → Timeline)
- FFmpeg 导出 (Timeline → MP4)
- QueuePage 集成 (显示生成进度)

### 成功指标

| 指标 | 目标 | 说明 |
|-----|------|------|
| 生成时间 | 15-30 分钟 | AI 并行生成 + 合成 |
| 生成成功率 | > 90% | 支持失败重试 |
| 视频质量 | 可直接发布 | 与专业编导接近 |
| 成本 | < ¥1.5/视频 | HeyGen + Claude |

---

## 🔧 实现计划 (5.5 周)

### Week 1: 模板库 + 参数 API (1 周)
**目标**: 模板和参数 API 就绪

- [ ] ShortVideoTemplate 数据模型 + 迁移
- [ ] ShortVideo 数据模型 + 迁移
- [ ] 2-3 个默认模板配置 (JSON)
- [ ] 模板 CRUD API (`/api/v1/short-video-templates`)
- [ ] 短视频 CRUD API (`/api/v1/short-videos`)
- [ ] 参数验证逻辑
- [ ] 单元测试 (API + 模型)

**交付**: 后端 API 可调用

### Week 2: HeyGen 虚拟主播集成 (1 周)
**目标**: 虚拟主播视频生成就绪

- [ ] HeyGen API 认证与连接
- [ ] HeyGen Provider Adapter (发送文案 → 获取视频)
- [ ] 多种虚拟主播角色选择
- [ ] 任务队列管理 (Job Orchestrator)
- [ ] 结果缓存 (避免重复调用)
- [ ] 错误处理与重试机制
- [ ] 单元测试

**交付**: 虚拟主播视频生成就绪

### Week 3: 文案 AI + 前端选择 (3 天)
**目标**: 文案自动生成 + 前端模板选择

- [ ] Claude API 集成
- [ ] 文案生成提示词模板
- [ ] 多语言文案支持
- [ ] TemplateSelector 前端组件
- [ ] 响应式样式
- [ ] 集成到 Studio 导航

**交付**: 模板选择 + 文案生成就绪

### Week 4: Timeline 合成 + 导出 (1 周)
**目标**: AI 结果 → Timeline → FFmpeg → MP4

- [ ] 参数 + AI 结果 → Timeline 转换逻辑
- [ ] ParametersPanel 前端组件
- [ ] PreviewPanel (进度跟踪)
- [ ] 导出 API (`POST /api/v1/short-videos/:id/export`)
- [ ] QueuePage 集成 (显示短视频任务)
- [ ] 导出完成通知
- [ ] 集成测试

**交付**: 导出就绪

### Week 5: 测试 + 优化 (0.5 周)
**目标**: MVP 生产就绪

- [ ] 单元测试 (所有 API)
- [ ] 集成测试 (参数 → 导出完整流程)
- [ ] 前端 e2e 测试 (选模板 → 导出)
- [ ] 性能优化 (缓存 / 并行处理)
- [ ] 错误处理补完
- [ ] 文档补完

**交付**: MVP 生产就绪 ✅

---

## 📚 技术决策

### 1. 为什么优先做虚拟主播（HeyGen）？
- ✅ 效果最自然 (真人级表演)
- ✅ 易于吸引用户 (创新感强)
- ✅ 可直接替代真人演员
- ✅ API 集成成熟

### 2. 为什么不做图像生成（v1）？
- 降低 MVP 复杂度
- 可用预设背景/产品图
- Comfy UI 留给 v2

### 3. 为什么用 Claude 而不是 GPT？
- 成本更低
- 质量够好
- 已在项目中使用

### 4. 为什么不做音乐生成？
- 用预设库 (免费, 成本低)
- Suno 太贵 (留给 v2)

---

## 📋 OUT OF SCOPE (v2+)

- Comfy UI 图像生成
- Suno 音乐生成
- 自定义模板编辑 UI
- 用户分段 / A/B 测试
- 分析数据面板

---

## 🎯 关键数字

- **MVP 工期**: 5.5 周
- **初期模板**: 2-3 个
- **成本**: < ¥1.5/视频
- **生成时间**: 15-30 分钟
- **成功率**: > 90%

---

## ✅ 验收标准

1. ✅ 用户选择模板 → 填参数 → 点导出 → 15-30 分钟内生成 MP4
2. ✅ 生成的视频可直接上传抖音/小红书
3. ✅ 后端 API 完全测试覆盖
4. ✅ 前端零 TypeScript 错误
5. ✅ QueuePage 能正常显示进度
6. ✅ 生成成功率 > 90%

