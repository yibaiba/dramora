# 角色一致性 L2/L3：IP-Adapter + LoRA 精细化控制

## Goal

在现有角色一致性系统（L1）基础上，增加 IP-Adapter 和 LoRA 微调能力，提升角色在多镜头生成中的一致性（面部特征、姿态、服装细节）。

目标：用户可以精细调整以下维度：
- **IP-Adapter 强度** (0-1.0) - 人物ID 参考影响程度
- **LoRA 权重** (0-1.0) - 细粒度特征学习强度
- **特征约束** - 锁定特定特征（眼睛、发型、肤色等）
- **角度/姿态适配** - 不同角度下的自动调整

## What I already know

**现有实现 (L1)**：
- CharacterBible 数据结构（Anchor, Palette, Expressions, ReferenceAssets, Wardrobe）
- ShotPromptPack 存储提示词（DirectPrompt, NegativePrompt）
- Seedance 图像生成提供商集成
- 生成参数预设 (preset: fast/balanced/quality)

**现有 API**：
- POST /api/v1/story-map-characters/{id}/character-bible:save
- GET /api/v1/storyboard-workspace/{episodeId}
- 提示词基于 ShotPromptPack 的 DirectPrompt 字段

**约束**：
- 图像生成通过 Seedance 提供商（未直接集成 IP-Adapter/LoRA）
- 提示词是字符串，非结构化
- L1 使用基础文本描述 + Reference Assets

## 关键决策 (已锁定 ✅)

| 决策项 | 选择 | 理由 |
|-------|------|------|
| **IP-Adapter/LoRA 来源** | Seedance 原生支持 | 简洁、最小化依赖 |
| **前端 UI 设计** | 简单 3-滑块 | MVP 快速交付（2-3 天） |
| **Provider 范围** | 多 Provider 支持 | 前瞻性，便于未来扩展 |
| **不支持时行为** | 显示禁用+提示 | 更好的用户体验和反馈 |

### 技术含义

**后端**：
- 无需集成新的推理引擎，Seedance API 直接支持
- 在提示词或生成参数中传入 ip_adapter_strength, lora_weight
- Provider 能力检测（检查当前 Provider 是否支持）

**前端**：
- ShotPromptPack 编辑面板新增 3 个滑块：
  - IP-Adapter 强度 (0-1.0)
  - LoRA 权重 (0-1.0)  
  - LoRA 组合权重 (0-1.0，可选多 LoRA)
- Provider 不支持时禁用 + 显示提示信息

**数据库**：
- ShotPromptPack 表新增 3 列（nullable）

## Requirements (MVP)

### 后端 (internal/)
* [ ] 数据库迁移：ShotPromptPack 表新增 3 列
  - ip_adapter_strength (FLOAT, default 0.5)
  - lora_weight (FLOAT, default 0.5)
  - lora_combination_weight (FLOAT, default 1.0)
* [ ] Domain 层新增验证：ValidateIPAdapterStrength() / ValidateLoRAWeight()
* [ ] Repository 层新增字段 save/load 逻辑
* [ ] Service 层在生成时应用这些参数到 Seedance 请求
* [ ] Provider 能力检测：检查 Provider 是否支持 IP-Adapter/LoRA
  - 如不支持，API 端禁用这些参数或返回 409 冲突

### 前端 (apps/studio/)
* [ ] ShotPromptPack 编辑面板新增 3 个滑块控制
  - IP-Adapter 强度 (0-1.0)
  - LoRA 权重 (0-1.0)
  - LoRA 组合权重 (0-1.0)
* [ ] Provider 不支持时：禁用滑块 + 显示提示信息 "该 Provider 不支持 IP-Adapter/LoRA"
* [ ] 参数实时保存（与其他 ShotPromptPack 字段同时保存）
* [ ] 参数验证：0-1.0 范围（前端 + 后端）

### 测试
* [ ] 单元测试：参数验证（有效/无效范围）
* [ ] 集成测试：保存 + 加载循环
* [ ] API 测试：Provider 能力检测

## Acceptance Criteria (MVP)

* [ ] ✅ IP-Adapter 强度 slider 显示并可调节
* [ ] ✅ LoRA 权重 slider 显示并可调节
* [ ] ✅ LoRA 组合权重 slider 显示并可调节
* [ ] ✅ Provider 不支持时禁用 + 提示
* [ ] ✅ 参数保存到数据库成功
* [ ] ✅ 生成时参数正确传递给 Seedance API
* [ ] ✅ 参数验证：0-1.0 范围检查
* [ ] ✅ TypeScript 零错误
* [ ] ✅ Go 零错误（gofmt + go test）
* [ ] ✅ 与现有 CharacterBible 和 ShotPromptPack 系统无缝集成

## Definition of Done

* Tests added for parameter validation and generation
* Lint / typecheck / CI green
* 前后端构建成功
* 文档更新
* 与现有一致性系统无缝集成

## Out of Scope (MVP)

* ❌ 多个 LoRA 模型库的完整管理（假设 Seedance 后端处理）
* ❌ 特征锁定（eyes, hair, costume 等细粒度控制）
* ❌ 角度/姿态适配自动调整
* ❌ 实时预览或对比生成
* ❌ 配置版本/预设保存
* ❌ IP-Adapter/LoRA 模型训练或微调

**后续版本**可考虑这些功能。

## 实现计划 (预估 2-3 周，3 个 PR)

### PR1: 后端基础 (2-3 天)
- ✅ 数据库迁移 + 字段
- ✅ Domain 层验证
- ✅ Repository 层 save/load
- ✅ Service 层集成（生成时应用参数）
- ✅ Provider 能力检测
- ✅ API 测试 + 单元测试

**预期提交**: 迁移 + 4 个 Go 文件修改

### PR2: API 端点 + 前端 Hook (2-3 天)
- ✅ HTTP handler 新增参数接收
- ✅ React Query hooks 类型定义 + API client
- ✅ 参数验证端点
- ✅ 集成测试

**预期提交**: 路由 + 2-3 个 TypeScript 文件

### PR3: 前端 UI + 完整集成 (2-3 天)
- ✅ ShotPromptPack 编辑面板新增滑块
- ✅ 禁用 + 提示信息逻辑
- ✅ 样式 CSS（Tailwind）
- ✅ 端到端测试

**预期提交**: QueuePage / StoryboardPage + CSS 修改

**总工期**: 6-9 天（可并行进行前端，总体 2 周内交付）

## Technical Approach

### 数据库架构
```sql
ALTER TABLE shot_prompt_packs ADD COLUMN (
  ip_adapter_strength FLOAT DEFAULT 0.5,
  lora_weight FLOAT DEFAULT 0.5,
  lora_combination_weight FLOAT DEFAULT 1.0
);
```

### API 约定
```
POST /api/v1/storyboard-shots/{shotId}/prompt-pack:save
Body: {
  directPrompt: string,
  negativePrompt: string,
  ipAdapterStrength: 0-1.0,
  loraWeight: 0-1.0,
  loraCombinationWeight: 0-1.0
}
```

### 前端状态管理
使用 React Query hooks + Zustand store
- useShotPromptPack() 查询当前参数
- useUpdatePromptPack() 保存参数
- 参数验证 hook：useValidateLoRAParams()

### Provider 能力检测
```go
// 后端 service 层
func (s *ProductionService) CanUseLoRA(providerType string) bool {
  // 检查 provider 是否支持 IP-Adapter/LoRA
  // Seedance: true, Others: false (待扩展)
}
```

## 技术债务与风险

1. **Seedance API 参数支持**
   - 需要确认 Seedance API 是否直接支持这些参数
   - 如不支持，需要通过提示词注入或适配层处理
   - **缓解**: 预留 adapter 接口，便于后续扩展

2. **Provider 能力检测**
   - 当前仅 Seedance，其他 Provider 自动禁用
   - **缓解**: 使用策略模式，便于新增 Provider 支持

3. **向后兼容性**
   - 新列默认值为 0.5（中等强度），不影响现有生成
   - **缓解**: 迁移脚本设置默认值
