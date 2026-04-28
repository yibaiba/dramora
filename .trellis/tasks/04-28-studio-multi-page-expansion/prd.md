# studio multi-page expansion

## Goal

为 Dramora Studio 从当前单页工作台扩展为多页面信息架构，先产出同风格页面视觉稿，再据此规划前后端实现范围，让后续开发能按 Trellis 任务流逐步落地。

## What I already know

* 当前前端是 `apps/studio` 下的 Vite + React + TypeScript 应用。
* 当前 Studio 没有路由，页面逻辑基本都集中在 `apps/studio/src/App.tsx`。
* 现有前端已经接通这些核心能力：项目、剧集、故事源、故事解析、资产图谱、候选资产、分镜卡、提示词包、审批点、生成队列、时间线、导出。
* 前端规范要求：服务端状态统一走 `src/api/hooks.ts`，DTO 统一走 `src/api/types.ts`，组件内不能直接 `fetch`。
* Studio 设计方向已经明确为专业暗色、电影感、AI 漫剧生产工作台，而不是普通后台。
* 已生成并保存在项目目录的参考视觉稿：
  * `apps/studio/public/ui-mockups/homepage-reference.png`
  * `apps/studio/public/ui-mockups/homepage-overview-variant.png`
  * `apps/studio/public/ui-mockups/homepage-storyboard-variant.png`
  * `apps/studio/public/ui-mockups/homepage-mobile-overview.png`
  * `apps/studio/public/ui-mockups/homepage-mobile-storyboard.png`

## Assumptions (temporary)

* 多页面改造会优先复用现有 hooks / DTO / mutations，而不是先重做 API。
* 第一阶段重点是信息架构和页面边界清晰化，不会一开始就把所有页面做成完整可编辑产品。
* 首页、故事解析、分镜、时间线/导出会是最优先的页面候选，因为现有前后端能力最完整。

## Open Questions

* 暂无。可进入 Storyboard 第二阶段实现。

## Requirements (evolving)

* 产出一组视觉风格一致的新页面 UI 稿。
* 页面范围要能映射到现有后端能力和现有 hooks。
* 后续实现规划要明确哪些页面先做前端结构，哪些页面需要补后端 API / DTO / mutation。
* 本轮 MVP 固定为 4 页：
  * 工作台首页
  * 故事源与解析
  * 分镜工作台
  * 时间线与导出
* 当前架构选择为“预留扩展位”：
  * 当前只实现 4 页
  * 但导航、路由常量、页面壳要为未来的“资产图谱 / 候选资产”独立页保留位置
  * 当前不提前做该页内容
* 当前进入第二阶段，优先完善 `StoryboardPage`
* 当前已确认 Storyboard 这一轮采用“补聚合读模型”的方向，而不是仅做前端产品化
* 当前已确认聚合读模型采用：
  * `GET /api/v1/episodes/{episodeId}/storyboard-workspace`
  * 写动作继续沿用现有 storyboard / prompt-pack / approval / generation POST routes

## Acceptance Criteria (evolving)

* [x] 明确 MVP 页面范围。
* [x] 明确是否为未来页面预留扩展位。
* [x] 明确每个页面对应的现有数据能力与缺口。
* [x] 明确后续前后端实现顺序。

## Definition of Done (team quality bar)

* Tests added/updated (unit/integration where appropriate)
* Lint / typecheck / CI green
* Docs/notes updated if behavior changes
* Rollout/rollback considered if risky

## Out of Scope (explicit)

* 本轮 brainstorm 不直接承诺完成整套多页面实现。
* 暂不把所有历史工作台模块一次性拆成成熟独立域。
* “资产图谱 / 候选资产”暂不作为独立 MVP 页面默认纳入。
* “项目 / 剧集总览”暂不作为独立 MVP 页面默认纳入。
* 默认不在这一轮就做 URL 深链，除非后续决策改为真实 URL 路由并把上下文同步进地址栏。

## Technical Notes

* 当前没有 `react-router` / `Routes` / `BrowserRouter`。
* 当前单页内已经具备的主要工作区块：
  * 首页/导演台总览
  * 故事源输入与多 Agent 解析
  * 资产图谱与生产流程提示
  * 分镜卡工作台与镜头检查器
  * 时间线与导出
* 当前已确定的 4 页 MVP 更适合先按现有能力重组为：
  * `Home`
  * `Story Analysis`
  * `Storyboard`
  * `Timeline / Export`
* 已确认：导航与页面壳要预留未来“Assets / Graph”页扩展位。
* 后续多页面拆分时，需要优先考虑：
  * 路由与 URL 状态如何承接当前 `selectedProjectId` / `selectedEpisodeId`
  * 是否拆分 `App.tsx` 为路由壳 + 页面级组件
  * 哪些页面只做读态展示，哪些页面保留写操作

## Research Notes

### Constraints from current repo

* 当前前端没有 `react-router`，也没有现成多页面目录结构。
* 现有能力集中在一个 `App.tsx` 中，说明拆页时必须先切出共享 Studio Shell。
* 已有 `selectedProjectId` 与 `selectedEpisodeId` 上下文，后续无论用真实路由还是本地切页，都需要一个稳定的共享上下文层。

### Feasible approaches here

**Approach A: React Router + Studio Shell** (recommended candidate)

* How it works:
  * 引入路由库
  * 拆出共享导航壳和 4 个页面组件
  * 为未来 Assets 页预留路由常量和导航位
* Pros:
  * 真正的多页面结构，后续扩展最顺
  * 页面级代码边界更清晰
* Cons:
  * 第一轮改动范围更大
  * 需要决定页面上下文如何跨路由保持

**Approach B: Shell 内部页面切换**

* How it works:
  * 暂不引入路由
  * 用共享布局 + 本地状态切换 4 个页面视图
  * 先把单页组件拆散
* Pros:
  * 改动更轻，短期交付快
* Cons:
  * 不是“真实页面”
  * 以后接路由时会再重构一次

## Technical Approach

采用 **React Router + Studio Shell**：

* 增加共享 `StudioShell`，承载左侧导航、项目/剧集上下文和未来页面扩展位
* 当前实现 4 个页面：
  * `HomePage`
  * `StoryAnalysisPage`
  * `StoryboardPage`
  * `TimelineExportPage`
* 在导航与路由常量中预留未来 `AssetsGraphPage` 入口，但当前不开放内容
* 先按现有 hooks 和 DTO 重组页面；只有在页面落地时发现接口粒度不够，才补后端 API / 聚合视图

### Page-to-capability mapping

**HomePage**

* Existing frontend data:
  * `useProjects`
  * `useEpisodes`
  * `useGenerationJobs`
  * `useStoryAnalyses`
  * `useStoryMap`
  * `useEpisodeAssets`
  * `useStoryboardShots`
  * `useEpisodeApprovalGates`
* Likely gaps:
  * 需要提炼首页聚合视图模型，减少页面自己拼装过多逻辑

**StoryAnalysisPage**

* Existing frontend data:
  * `useStorySources`
  * `useCreateStorySource`
  * `useStartStoryAnalysis`
  * `useStoryAnalyses`
* Likely gaps:
  * 后端能力基本够用，主要是前端页面拆分与信息结构重组

**StoryboardPage**

* Existing frontend data:
  * `useStoryMap`
  * `useEpisodeAssets`
  * `useEpisodeApprovalGates`
  * `useGenerationJobs`
  * `useStoryboardShots`
  * `useUpdateStoryboardShot`
  * `useGenerateShotPromptPack`
  * `useSaveShotPromptPack`
  * `useStartShotVideoGeneration`
* Likely gaps:
  * 当前页面需要自行拼装很多查询结果，前端编排负担较重
  * 现有 Storyboard API 已覆盖核心 happy path，但缺少“工作台聚合读模型”
  * `request-changes` 等审批动作后端已存在，但页面层还可以继续产品化

**TimelineExportPage**

* Existing frontend data:
  * `useEpisodeTimeline`
  * `useSaveEpisodeTimeline`
  * `useStartEpisodeExport`
  * `useExport`
* Likely gaps:
  * 后端能力已基本具备，主要是页面独立化和导出状态展示的 UX 重构

## Decision (ADR-lite)

**Context**: 当前 Studio 已经拥有足够多的生产能力，但全部挤在单页 `App.tsx` 中，不利于继续扩展页面和补充后端能力。

**Decision**: 采用 `React Router + Studio Shell`，做 4 页 MVP，并在导航/路由结构中预留未来 Assets 页扩展位。

**Consequences**:

* 第一轮前端重构量会大于“内部切页”方案
* 但页面边界、后续 API 对齐、以及继续扩图/扩功能的成本会更低
* 初期以前端结构重组为主，后端只补真正暴露出的页面数据缺口

## Storyboard Phase Research

### Current reality

* 已有核心路线：
  * 镜头列表：`GET /api/v1/episodes/{episodeId}/storyboard-shots`
  * 镜头更新：`POST /api/v1/storyboard-shots/{shotId}:update`
  * 提示词包读取/生成/保存：
    * `GET /api/v1/storyboard-shots/{shotId}/prompt-pack`
    * `POST /api/v1/storyboard-shots/{shotId}/prompt-pack:generate`
    * `POST /api/v1/storyboard-shots/{shotId}/prompt-pack:save`
  * 视频生成：`POST /api/v1/storyboard-shots/{shotId}/videos:generate`
  * 审批点读取/seed/approve/request-changes` 已具备
* 说明：后端并不是“缺功能”，而是前端页面为了组成完整工作台，需要同时拉很多 query。

### Feasible approaches for Storyboard next

**Approach A: Existing-API Productization**

* How:
  * 保持现有 API 不变
  * 继续把 Storyboard 页面做实：完善审批动作、生成状态、提示词编辑流、空态/禁用态、从分镜到时间线的交互
* Pros:
  * 风险最低，最快进入可用产品页
  * 不改后端读模型
* Cons:
  * 前端仍需拼装较多 query

**Approach B: Storyboard Workspace Read Model** (recommended candidate)

* How:
  * 新增一个聚合读取接口，例如 episode 级 storyboard workspace 视图
  * 一次返回镜头、审批、资产摘要、生成队列摘要、提示词包就绪态等工作台必需信息
  * 前端 StoryboardPage 改为以聚合读模型为主，写动作仍复用现有 POST routes
* Pros:
  * 页面数据流更稳，前端更轻
  * 更适合后续继续扩分镜工作台
* Cons:
  * 需要补后端 API / DTO / hooks

**Approach C: Extended Storyboard Workflow**

* How:
  * 在方案 B 基础上，再加更深的工作流动作，例如更细的人审/送时间线/更多镜头级状态
* Pros:
  * 更接近完整生产系统
* Cons:
  * 范围明显变大，不适合这一轮先手

## Storyboard Endpoint Shape Research

### Repo constraints

* 当前前端-facing API 明确遵循 **GET for reads / POST for writes**。
* `internal/httpapi/router.go` 中 episode 级读接口都采用 `/api/v1/episodes/{episodeId}/<resource>` 风格。
* 现有写路由已经足够，不需要为了聚合读模型去改动 `POST /storyboard-shots/*` 系列动作。

### Feasible endpoint shapes

**Approach 1: New GET `/api/v1/episodes/{episodeId}/storyboard-workspace`** (recommended candidate)

* How:
  * 新增一个 episode 级聚合读接口
  * 返回 Storyboard 页面主数据：shots、approval summary、asset summary、generation job summary、prompt-pack readiness 等
* Pros:
  * 语义最清晰
  * 不污染现有 `storyboard-shots` 资源定义
  * 前端 StoryboardPage 可以直接以“工作台视图模型”为主
* Cons:
  * 需要新增 DTO / handler / service read model

**Approach 2: Enrich existing GET `/api/v1/episodes/{episodeId}/storyboard-shots`**

* How:
  * 在当前 shots list 响应里塞入 workspace summary
* Pros:
  * 少一个新路由
* Cons:
  * 语义混杂：镜头列表接口不再只是镜头列表
  * 后续复用 storyboard_shots 资源时会更笨重

**Approach 3: New GET `/api/v1/episodes/{episodeId}/storyboard-overview` + keep shots list**

* How:
  * 增加一个 overview 接口，只返回摘要
  * shots 仍单独拉 list
* Pros:
  * 比方案 2 清晰
* Cons:
  * 前端仍需要至少两次请求拼装工作台

## Storyboard Phase Decision (ADR-lite)

**Context**: 当前 StoryboardPage 的核心动作链路已经存在，但页面需要同时拼装多个 query，导致前端负担重、页面级模型不清晰。

**Decision**: 新建 `GET /api/v1/episodes/{episodeId}/storyboard-workspace`，作为 Storyboard 页的聚合读模型；现有写接口保持不变。

**Consequences**:

* 需要补 OpenAPI、后端 DTO/read model、handler/service、前端 client/hooks/types
* StoryboardPage 的主数据来源将更稳定，后续功能扩展更顺
* 粒度写操作仍保持资源化，不把所有编辑动作塞进一个大接口

## Current Status

* [x] 4 页 Studio MVP 已完成：`Home`、`Story Analysis`、`Storyboard`、`Timeline / Export`
* [x] `StoryboardPage` 已切到 `GET /api/v1/episodes/{episodeId}/storyboard-workspace` 作为主读模型
* [x] 写动作继续沿用现有 `storyboard-shots` / `prompt-pack` / `approval-gates` / `videos:generate` POST contracts
* [x] 已补充未播种 episode 的 workspace 空态回归测试，明确该接口返回 `200 + empty workspace`
* [x] `StoryboardPage` 已补充审批状态看板、逐条 approval gate 操作、送入 Timeline handoff 与提示词快捷插入
* [x] 预留的 `Assets / Graph` 页面已启用，并实现了故事图谱总览、候选资产池与锁定参考资产动作
* [x] 前后端当前验证已通过：Go test/build + Studio lint/build

## Recommended Next Step

* 下一轮更适合补审批闭环（`changes_requested -> pending`）、深化 `Assets / Graph` 的联动筛选/批量操作，或让 `Timeline / Export` 更直接消费已保存的 timeline 数据。

---

---

# 多 Agent 协作分析优化方案

> 基于 GitHub 开源项目调研（2026-04-28），为 dramora 的 AI 漫剧多 Agent 协作分析提供四阶段演进方案。

## 调研摘要

### 调研对象

**通用多 Agent 框架：**
- OATS — 生产级多 Agent 编排，Blackboard 协议、信仼评分、模型分层
- Harmonist — 机械门禁强制执行，供应链验证
- Squad (GitHub/Microsoft) — Drop-box 记忆、独立审查协议
- LangGraph — DAG 状态机编排，检查点/恢复，人工节点
- CrewAI — 角色-任务-团队三元模型，YAML 配置
- AutoGen (Microsoft) — 对话式多 Agent 协作
- MetaGPT — 虚拟软件公司 SOP 模式

**AI 影视/动画制作专项：**
- AniME (Bilibili) — Director Agent + 7 个专业 Agent + MCP 工具选择
- AniMaker (阿里+哈工大) — MCTS 驱动片段生成，节省 51% 计算
- MovieAgent (ShowLab) — 多 LLM Agent 模拟电影制作全流程
- MM-StoryAgent — 多模态叙事视频生成（文本+图像+音频）

**关键参考开源仓库：**
- [github.com/showlab/MovieAgent](https://github.com/showlab/MovieAgent) — 最完整的电影级多 Agent 框架
- [github.com/Pattyboi101/oats-autonomous-agents](https://github.com/Pattyboi101/oats-autonomous-agents) — Blackboard 协议参考实现
- [github.com/GammaLabTechnologies/harmonist](https://github.com/GammaLabTechnologies/harmonist) — 机械门禁参考
- [github.com/langchain-ai/langgraph](https://github.com/langchain-ai/langgraph) — DAG 状态机编排参考

### 当前 dramora 状态

| 维度 | 当前状态 | 差距 |
|------|---------|------|
| Agent 执行 | `story_analyzer.go` 正则/关键词确定性模拟 5 个 Agent | 无真实 LLM 调用 |
| Agent 数量 | 5 个模拟 Agent | 蓝图 15 个，10 个未实现 |
| 编排引擎 | `workflow/graph.go` 仅定义常量，未实际驱动执行 | DAG 未接入运行时 |
| 状态管理 | DB 读写，无检查点 | 无法断点恢复 |
| LLM Provider | 仅 Seedance（豆包视频生成） | 无 LLM Provider |
| 成本控制 | 无 | 无预算/分层 |
| 可观测性 | 基础日志 | 无追踪/重放 |

### 从开源项目提炼的核心模式

1. **Blackboard 协议 (OATS)**：Agent 不直接通信，读写共享状态面。复杂任务上比层级模式提升 13-57%
2. **DAG + 人工审批节点 (LangGraph/AniME)**：确定性流水线 + Human-in-the-Loop
3. **模型分层 (OATS)**：便宜模型做 80% 常规工作，贵模型做关键决策，节省 60-70% token
4. **机械门禁 (Harmonist)**：状态机 + 钩子强制规则，而非依赖 prompt 请求
5. **独立审查 (Squad)**：原作者不能修订自己被驳回的产出，必须换 Agent
6. **MCTS 驱动生成 (AniMaker)**：候选片段 + 评估选择，减少无效生成

---

## 架构决策 (ADR-lite)

### 决策 1: Provider 无关抽象层

**Context**: 当前仅接入 Seedance 视频生成，分析类 Agent 使用确定性模拟。需要引入 LLM 支持但不应锁定单一供应商。

**Decision**: 扩展现有 `internal/provider/adapter.go` 的 `Adapter` 接口，增加 LLM TaskType 和对应的 Adapter 实现。与 Seedance Adapter 同级，采用相同的可插拔模式。

**Consequences**:
- 新增 `LLMAdapter` 接口（或扩展现有 `Adapter`）
- 支持 OpenAI、Anthropic、DeepSeek、豆包 等多 Provider
- 通过环境变量/配置切换 Provider，无需改代码
- Agent 通过统一接口调用 LLM，不感知底层 Provider

### 决策 2: 混合编排（DAG + Blackboard）

**Context**: 漫剧生产天然分两类 Agent — 生产流水线（有明确上下游依赖）和审查优化（事件驱动，需随时检查任意阶段产出）。

**Decision**: 核心生产流水线采用 DAG 状态机保证确定性；审查/优化 Agent 通过 Blackboard 共享状态松耦合协作。

**Consequences**:
- DAG 负责：Story Analysis → Screenwriter → Character/Scene/Prop Design → Storyboard → Prompt → Video Gen → Editor → Export
- Blackboard 负责：Continuity Supervisor、Safety & Copyright、Cost Controller、Director 可随时读取任意阶段产出并注入审查结果
- Character/Scene/Prop 三个 Designer 可并行执行（节省时间）
- 人工审批节点（ApprovalGate）插入 DAG 固定位置

### 决策 3: 四阶段分批实施

**Context**: 范围大，需控制风险，优先验证架构可行性。

**Decision**: 分四阶段推进，Phase 1 最小范围验证架构，后续逐步扩展。

**Consequences**: Phase 1 仅替换现有 5 Agent，不做新 Agent，确保架构可工作后再扩展。

---

## 四阶段演进路线

### Phase 1: LLM 基础设施 + 现有 Agent 升级

**目标**: 搭建 Provider 抽象层 + DAG 引擎 + Blackboard 协议，替换现有 5 个确定性 Agent 为真实 LLM Agent

**范围**:

| 模块 | 变更 | 说明 |
|------|------|------|
| `internal/provider/` | 新增 `llm_adapter.go` | LLM Adapter 接口 |
| `internal/provider/` | 新增 `deepseek_adapter.go` | DeepSeek 内置实现（默认 Provider） |
| `internal/provider/` | 新增 `openai_compatible_adapter.go` | **通用 OpenAI 兼容适配器**（见下方详细设计） |
| `internal/workflow/` | 扩展 `graph.go` | DAG 运行时引擎：节点执行、边触发、状态转换、检查点 |
| `internal/workflow/` | 新增 `blackboard.go` | Blackboard 共享状态协议：读写锁、变更通知、订阅 |
| `internal/domain/` | 扩展 `production.go` | AgentRun/AgentStep/AgentArtifact 实体完善 |
| `internal/service/` | 重写 `story_analyzer.go` | 5 个 Agent 从确定性模拟改为 LLM Agent |
| `internal/service/` | 新增 `agent_service.go` | Agent 生命周期管理、Prompt 模板、输出解析 |
| `internal/httpapi/` | 新增 Agent 相关 DTO | AgentRun/AgentStep 的 API 响应 |

**端点设计（纯管理员配置，无内置 Provider）**:

> 核心原则：**不存在"内置 Provider"**。所有 AI 能力（对话/生图/视频/语音）全部通过管理员配置的端点接入。
> 管理员不配 → 系统不可用，给出明确提示。管理员配置了什么就用什么。

```
端点架构（纯配置驱动）:
  ┌──────────────────────────────────────────────────────────┐
  │                    管理员环境变量                           │
  │                                                           │
  │  # 每个能力独立配置端点（管理员想用什么就用什么）            │
  │  PROVIDER_CHAT_BASE_URL=https://ai-gateway.com/v1         │
  │  PROVIDER_CHAT_API_KEY=sk-xxx                             │
  │  PROVIDER_CHAT_MODEL=deepseek-chat                        │
  │  PROVIDER_CHAT_CREDITS_PER_CALL=5                         │
  │                                                           │
  │  PROVIDER_IMAGE_BASE_URL=https://ai-gateway.com/v1        │
  │  PROVIDER_IMAGE_API_KEY=sk-xxx                            │
  │  PROVIDER_IMAGE_MODEL=flux-pro                            │
  │  PROVIDER_IMAGE_CREDITS_PER_CALL=10                       │
  │                                                           │
  │  PROVIDER_VIDEO_BASE_URL=https://ark.volces.com/api/v3    │
  │  PROVIDER_VIDEO_API_KEY=ark-xxx                           │
  │  PROVIDER_VIDEO_MODEL=doubao-seedance-1-0-pro             │
  │  PROVIDER_VIDEO_CREDITS_PER_SECOND=20                     │
  │                                                           │
  │  PROVIDER_AUDIO_BASE_URL=https://ai-gateway.com/v1        │
  │  PROVIDER_AUDIO_API_KEY=sk-xxx                            │
  │  PROVIDER_AUDIO_MODEL=qwen-tts                            │
  │  PROVIDER_AUDIO_CREDITS_PER_CHAR=1                        │
  └──────────────┬───────────────────────────────────────────┘
                 │
                 ▼
  ┌──────────────────────────────────────────────────────────┐
  │              统一适配器（纯 OpenAI 兼容协议）               │
  │                                                           │
  │  Chat Completion    → POST {chat_base_url}/chat/completions│
  │  Image Generation   → POST {image_base_url}/images/gen    │
  │  Video Generation   → POST {video_base_url}/...           │
  │  Audio TTS          → POST {audio_base_url}/audio/speech  │
  │                                                           │
  │  每个能力可指向不同的 base_url（或同一个网关）              │
  │  未配置的能力 → 启动时报错，提示管理员配置                   │
  └──────────────────────────────────────────────────────────┘
```

**配置方式（前端管理后台，存数据库，即时生效）**:

> 端点配置不走环境变量。管理员在 Studio 管理后台页面配置，存入 DB，即时生效，无需重启。

```
配置流程:
  ┌──────────┐    POST /api/v1/admin/providers    ┌──────────┐
  │ 管理员     │  ──────────────────────────────>   │ 后端      │
  │ Studio    │                                     │ 存入 DB   │
  │ 管理后台   │  <──────────────────────────────   │ 即时生效   │
  └──────────┘    返回当前配置                       └──────────┘
```

**数据库存储（`provider_configs` 表）**:

```sql
CREATE TABLE provider_configs (
    id          TEXT PRIMARY KEY,
    capability  TEXT NOT NULL,        -- chat | image | video | audio
    base_url    TEXT NOT NULL,
    api_key     TEXT NOT NULL,        -- 加密存储
    model       TEXT NOT NULL,
    credits_per_unit INT NOT NULL,    -- 积分单价
    credit_unit TEXT NOT NULL,        -- per_call | per_second | per_char
    timeout_ms  INT DEFAULT 120000,
    max_retries INT DEFAULT 3,
    is_enabled  BOOLEAN DEFAULT true,
    updated_at  TIMESTAMP,
    updated_by  TEXT                  -- 操作管理员 ID
);
```

**前端管理后台页面**:

```
AdminSettingsPage (/admin/settings)
  ├─ 端点配置卡片
  │   ├─ LLM 对话端点
  │   │   ├─ base_url:  [https://your-gateway.com/v1        ]
  │   │   ├─ api_key:   [sk-xxxxxxxxxxxxxxxx                ] [👁]
  │   │   ├─ model:     [deepseek-chat                      ]
  │   │   ├─ 积分:      [5    ] 每次调用
  │   │   └─ [测试连接]  [保存]
  │   │
  │   ├─ 图像生成端点
  │   │   ├─ base_url:  [https://your-gateway.com/v1        ]
  │   │   ├─ api_key:   [sk-xxxxxxxxxxxxxxxx                ] [👁]
  │   │   ├─ model:     [flux-pro                           ]
  │   │   ├─ 积分:      [10   ] 每次调用
  │   │   └─ [测试连接]  [保存]
  │   │
  │   ├─ 视频生成端点
  │   │   └─ ...
  │   │
  │   └─ TTS 语音端点
  │       └─ ...
  │
  ├─ BILLING_MODE 开关
  │   ○ OFF      ○ SHADOW（记账不扣费）  ● ENFORCE（正式扣费）
  │
  └─ 积分套餐管理
      ├─ 免费体验: 200 积分 / ¥0
      ├─ 创作者:   2,000 积分/月 / ¥29
      ├─ 专业版:   5,000 积分/月 / ¥69
      └─ 工作室:   20,000 积分/月 / ¥199
```

**适配器实现**:

```go
// internal/provider/adapter.go

// ProviderSet 管理员配置的全部端点
type ProviderSet struct {
    Chat  ChatProviderConfig   // LLM 对话端点
    Image ImageProviderConfig  // 生图端点
    Video VideoProviderConfig  // 视频端点
    Audio AudioProviderConfig  // TTS 端点
}

type ChatProviderConfig struct {
    BaseURL          string
    APIKey           string
    Model            string
    CreditsPerCall   int
    Timeout          time.Duration
    MaxRetries       int
}

type ImageProviderConfig struct {
    BaseURL          string
    APIKey           string
    Model            string
    CreditsPerCall   int
    Timeout          time.Duration
    MaxRetries       int
}

type VideoProviderConfig struct {
    BaseURL          string
    APIKey           string
    Model            string
    CreditsPerSecond int  // 按秒计积分
    Timeout          time.Duration
    MaxRetries       int
}

type AudioProviderConfig struct {
    BaseURL          string
    APIKey           string
    Model            string
    CreditsPerChar   int  // 按字计积分
    Timeout          time.Duration
    MaxRetries       int
}

// LoadProviderSet 从数据库加载端点配置
// 配置由管理员在 Studio 管理后台维护，即时生效
func LoadProviderSet(ctx context.Context, repo ProviderConfigRepository) (*ProviderSet, error) {
    configs, err := repo.ListEnabled(ctx)
    if err != nil {
        return nil, fmt.Errorf("加载端点配置失败: %w", err)
    }
    // 逐一加载 chat/image/video/audio 配置
    // 未配置的能力 → 该能力不可用，前端页面给出提示
}
```

```go
// internal/provider/openai_compatible_adapter.go

// UnifiedAdapter 统一适配器
// 所有能力通过管理员配置的端点调用，不存在内置 Provider
type UnifiedAdapter struct {
    providers  *ProviderSet
    httpClient *http.Client
}

func NewUnifiedAdapter(ps *ProviderSet) *UnifiedAdapter {
    return &UnifiedAdapter{
        providers:  ps,
        httpClient: &http.Client{Timeout: ps.Chat.Timeout},
    }
}

func (a *UnifiedAdapter) ChatCompletion(
    ctx context.Context, req ChatRequest,
) (ChatResponse, error) {
    cfg := a.providers.Chat
    return a.doChat(ctx, cfg, req)
}

func (a *UnifiedAdapter) GenerateImage(
    ctx context.Context, req ImageRequest,
) (ImageResponse, error) {
    cfg := a.providers.Image
    return a.doImage(ctx, cfg, req)
}

func (a *UnifiedAdapter) GenerateVideo(
    ctx context.Context, req VideoRequest,
) (VideoResponse, error) {
    cfg := a.providers.Video
    return a.doVideo(ctx, cfg, req)
}

func (a *UnifiedAdapter) GenerateSpeech(
    ctx context.Context, req TTSRequest,
) (TTSResponse, error) {
    cfg := a.providers.Audio
    return a.doTTS(ctx, cfg, req)
}
```

**积分计算流程（纯端点配置）**:

```
用户触发操作
  │
  ├─ BILLING_MODE=OFF → 不检查积分，直接执行
  │
  └─ BILLING_MODE=SHADOW/ENFORCE
      │
      ├─ 查询积分单价（来自管理员配置的环境变量）:
      │   ├─ LLM 调用   → PROVIDER_CHAT_CREDITS_PER_CALL
      │   ├─ 生图       → PROVIDER_IMAGE_CREDITS_PER_CALL
      │   ├─ 视频       → PROVIDER_VIDEO_CREDITS_PER_SECOND × duration
      │   └─ TTS        → PROVIDER_AUDIO_CREDITS_PER_CHAR × text_length
      │
      ├─ SHADOW: 记录流水(amount=0) + 标记 capability + model
      └─ ENFORCE: 扣积分(amount=-N) + 标记 capability + model
```

**管理员可见的积分统计**:

| 维度 | 示例 |
|------|------|
| 自定义端点 LLM 调用 | 1,200 次 · 消耗 6,000 积分 |
| 自定义端点生图 | 80 次 · 消耗 800 积分 |
| 内置 Seedance 视频 | 12 次 · 消耗 2,400 积分 |
| 内置 DeepSeek LLM | 50 次 · 消耗 250 积分 |

**Agent 升级详情**:

```
现有（确定性）                     →  升级后（LLM Agent）
─────────────────────────────────────────────────────
story_analyst:    关键词匹配       →  LLM 深度分析主题/冲突/主线
outline_planner:  4句等分          →  LLM 识别剧情节拍/情节点
character_analyst: 关键词提取 2-6字 →  LLM 角色画像/关系/动机
scene_analyst:    取前4句前12字     →  LLM 场景识别/氛围/视觉元素
prop_analyst:     硬编码关键词列表  →  LLM 道具识别/用途/场景关联
```

**不在此阶段**:
- 不新增 Screenwriter、Director 等其余 10 个 Agent
- 不做模型分层/成本控制
- 不做 Web 剪辑功能

### Phase 2: 多 Agent 编排引擎完善 + 核心新 Agent

**目标**: DAG 引擎真正驱动全流程，补充 Screenwriter/Director/Cinematographer/Voice Agent

**范围**:

| 新增 Agent | 职责 | 依赖 |
|-----------|------|------|
| Screenwriter | 故事→剧集脚本/场景分解/对白/旁白 | Story Analysis 产出 |
| Director | 视觉连续性/镜头路线/关键帧规划 | Storyboard + Character/Scene |
| Cinematographer | 镜头语言优化/构图/机位/灯光 | Storyboard Shots |
| Voice & Subtitle | TTS 脚本/字幕片段/配音风格 | Screenwriter 对白产出 |

**基础设施**:
- DAG 检查点/恢复机制
- 并行 Agent 执行（Character/Scene/Prop Designer 可并行）
- Agent 间共享上下文传递优化

### Phase 3: 质量与安全体系

**目标**: 独立审查 Agent、安全合规、模型分层降本

**范围**:

| 新增 Agent | 职责 |
|-----------|------|
| Continuity Supervisor | 角色/场景/道具一致性检查、时间线错误检测 |
| Safety & Copyright | 内容安全审查、版权风险筛查 |
| Cost Controller | 按 Agent 预算管理、模型分层路由 |

**模型分层策略** (参考 OATS):

```
便宜模型 (DeepSeek):        贵模型 (Claude/Opus):
├─ Story Analyst            ├─ Director（关键创意决策）
├─ Outline Planner          ├─ Continuity Supervisor（质量把关）
├─ Character/Scene/Prop     ├─ Safety & Copyright（合规硬门禁）
├─ Voice & Subtitle         └─ 人工审批后的重生成
├─ Cinematographer
└─ 常规 Prompt Engineering
```

### Phase 4: 生产级优化 + Web 剪辑

**目标**: OTel 可观测性、Web 页面剪辑、成本追踪、JSONL 重放调试

**范围**:

| 功能 | 说明 |
|------|------|
| OTel 可观测性 | Agent 调用链追踪、耗时/Token 统计、错误归因 |
| Web 剪辑功能 | TimelineExportPage 增加剪辑能力：片段裁剪、转场、字幕对齐、预览 |
| JSONL 重放 | Agent 执行记录可回放，用于调试和回归验证 |
| 并行优化 | 非依赖 Agent 并行执行，减少端到端延迟 |
| Cost Ledger | 按项目/剧集/Agent 维度核算成本 |

---

## 关键设计原则（来自开源项目）

### 1. 机械门禁 (Harmonist)

审查 Agent 的结论不是"建议"，而是硬阻断：
- Safety Agent 标记 `block` → DAG 状态机必须停止该分支
- Continuity Supervisor 发现角色漂移 → 自动创建 `ReviewIssue`，阻塞后续生成
- 人工审批节点未通过 → 下游 Agent 不可触发

### 2. 独立审查 (Squad)

- 被驳回的产出由**不同的 Agent** 修复，而非原 Agent 自我修正
- 例如：Director 驳回 Storyboard Agent 的镜头规划 → Screenwriter Agent 重新输出，Storyboard Agent 重新生成

### 3. 共享状态不可变追加 (Blackboard)

- Agent 不修改其他 Agent 的产出，只追加自己的分析
- 所有 Agent 产出带版本号和签名
- 冲突通过 Director Agent 或人工审批裁决

### 4. 每个 Agent 产出结构化数据，非自由文本

- Agent 输出必须是 `domain.StoryAgentOutput` 结构，包含结构化字段
- 禁止 Agent 输出"自由格式 Markdown 供下游 Agent 自行理解"
- 下游 Agent 通过类型安全的字段读取上游产出

---

## Phase 1 实现计划 (PR 拆分)

```
PR1: provider — 统一端点适配器（DB 配置驱动，管理后台维护）
  ├─ internal/provider/adapter.go: ProviderSet + 各能力配置结构体
  ├─ internal/provider/unified_adapter.go: UnifiedAdapter (chat/image/video/audio)
  ├─ internal/provider/config_repo.go: ProviderConfigRepository 接口 + SQLite/Postgres 双实现
  ├─ internal/httpapi/admin_providers.go: 管理后台配置 API
  └─ test: adapter 单元测试 + mock 端点

PR1.5: repo/sqlite — SQLite 持久化（默认数据库）
  ├─ internal/repo/sqlite.go: OpenSQLite + RunMigrations (WAL 模式 + 外键约束)
  ├─ internal/repo/migrations/: 所有表的建表语句（与 PostgreSQL 结构一致）
  └─ internal/app/container.go: 扩展 → DATABASE_URL 未配置时默认走 SQLite

PR2: workflow/engine — DAG 运行时 + Blackboard 协议
  ├─ internal/workflow/engine.go: DAG 执行引擎
  ├─ internal/workflow/blackboard.go: Blackboard 共享状态
  ├─ internal/workflow/checkpoint.go: 检查点/恢复
  └─ test: 引擎单元测试

PR3: domain + service — Agent 实体 + 5 Agent 升级
  ├─ internal/domain/production.go: AgentRun/AgentStep/AgentArtifact
  ├─ internal/service/agent_service.go: Agent 生命周期/Prompt 模板
  ├─ internal/service/story_analyzer.go: LLM Agent 替换确定性模拟
  └─ test: Agent 集成测试

PR4: httpapi + Studio — API + 前端 Agent 进度展示
  ├─ internal/httpapi/agent_dto.go: Agent DTO
  ├─ internal/httpapi/agent_routes.go: Agent 相关路由
  ├─ apps/studio/src/api/: Agent hooks/types
  └─ StoryAnalysisPage: Agent 执行进度/产出展示
```

---

## 决策汇总

| # | 决策 | 选项 |
|---|------|------|
| 1 | 实施节奏 | 全四阶段规划，分批实施 |
| 2 | AI 端点 | **前端管理后台配置，存 DB 即时生效；4 能力独立端点 + 积分单价，无内置 Provider** |
| 3 | 编排模型 | 混合：DAG（生产流水线）+ Blackboard（审查层） |
| 4 | Phase 1 范围 | 基础设施 + 仅替换现有 5 Agent |
| 5 | Web 剪辑 | 纳入 Phase 4，TimelineExportPage 增强 |

---

## 验收标准

### Phase 1 验收

- [ ] LLM Adapter 接口定义并可切换 Provider
- [ ] DAG 引擎可执行简单流水线（≥3 个节点的串行 DAG）
- [ ] Blackboard 协议支持多 Agent 读写共享状态
- [ ] 5 个 Agent 全部从确定性模拟切换为 LLM Agent
- [ ] AgentRun/AgentStep/AgentArtifact 正确持久化
- [ ] Studio StoryAnalysisPage 展示真实 Agent 执行进度
- [ ] `story_analyzer.go` 中的 `analyzeStorySource()` 等确定性函数标记为 deprecated
- [ ] Go test + Studio lint/build 通过
- [ ] OpenAPI 更新新增/变更的 API contract

### Phase 2 验收

- [ ] DAG 检查点/恢复可用（中断后可从断点继续）
- [ ] Screenwriter/Director/Cinematographer/Voice Agent 全部可用
- [ ] Character/Scene/Prop Designer 并行执行验证通过
- [ ] 端到端：故事输入 → 分镜产出 全链路 LLM 驱动

### Phase 3 验收

- [ ] Continuity Supervisor 可检测角色/场景/道具不一致
- [ ] Safety Agent 可阻断不安全内容
- [ ] 模型分层路由生效（便宜模型承担 ≥60% 调用量）
- [ ] Cost Controller 按 Agent 维度输出成本报表

### Phase 4 验收

- [ ] Web 剪辑功能：片段裁剪、转场、字幕对齐、预览
- [ ] OTel 追踪覆盖所有 Agent 调用
- [ ] JSONL 重放可复现 Agent 执行结果
- [ ] 端到端延迟相比 Phase 1 降低 ≥30%（通过并行执行）

---

---

# 第二轮调研：AI 漫剧竞品 + Web 剪辑方案

> 调研日期：2026-04-28

## 竞品分析 — AI 漫剧生产工具

### 直接竞品

#### 1. waoowaoo（哇哦哇哦）⭐ 10K+

| 维度 | 详情 |
|------|------|
| **定位** | "首家工业级全流程 AI 影视生产平台" |
| **GitHub** | [waoowaooAI/waoowaoo](https://github.com/waoowaooAI/waoowaoo) |
| **技术栈** | Next.js 15 + React 19 + PostgreSQL + Prisma + Redis + BullMQ + Tailwind CSS v4 |
| **AI 服务** | Vercel AI SDK + OpenRouter + Google GenAI + fal.ai |
| **流程** | 小说文本 → AI 剧本分析 → 角色/场景一致性生成 → 分镜视频 → AI 多角色配音 → 完整视频输出 |
| **商业化** | SHADOW 三态计费（OFF/SHADOW/ENFORCE）+ Capability 组合定价 |
| **架构** | Next.js 单体 + 4 独立 Worker（image/video/voice/text）+ BullMQ 任务队列 |
| **关键亮点** | 多用户 SaaS 体系、计费系统完备、Docker 一键部署 |

**dramora 对比优势：**
- dramora 的 DAG + Blackboard 编排模型比 waoowaoo 的固定 Worker 更灵活
- dramora 的 Provider 抽象层比 waoowaoo 直接耦合 AI SDK 更易扩展
- dramora 的 ApprovalGate 体系更完善（waoowaoo 无人审流程）
- dramora 有完整的故事图谱/资产图谱，waoowaoo 只有简单角色/场景管理

**dramora 可借鉴：**
- BullMQ 任务队列模式（waoowaoo 的 DB ↔ BullMQ 三层对账机制防止僵尸任务）
- Capability 组合定价思路（模型 × 分辨率 × 时长 × 模式）
- 独立 Worker 进程模式（按任务类型拆分 Worker）

#### 2. Openframe ⭐ 较新

| 维度 | 详情 |
|------|------|
| **定位** | "开源免费的 AI 漫剧创作工作台" |
| **GitHub** | [murongg/openframe](https://github.com/murongg/openframe) |
| **协议** | AGPL-3.0 |
| **技术栈** | 客户端优先（Web + Desktop），无服务端后端依赖 |
| **流程** | 剧本编辑+AI辅助 → 角色/道具/场景管理 → 人物关系图谱 → 分镜生成 → 时间线编排 → FCPXML/EDL 导出 |
| **关键亮点** | 人物关系图谱、FCPXML/EDL 导出（可接入 PR/达芬奇）、双端支持 |

**dramora 对比优势：**
- dramora 是真正的 SaaS 多用户平台，Openframe 是单机工具
- dramora 有完整的后端 API + 数据库 + 异步任务系统
- dramora 的视频生成链路（Seedance 集成）比 Openframe 更成熟

**dramora 可借鉴：**
- **人物关系图谱** — 比 dramora 当前的 StoryMap 更可视化、更交互化
- **FCPXML/EDL 导出** — 让用户在 dramora 完成粗剪后，可导出到 PR/达芬奇精剪
- **剧本编辑器 + AI 辅助** — 不仅是文本解析，还提供补全/润色/改编等编辑器内 AI 能力

#### 3. creative_tools

| 维度 | 详情 |
|------|------|
| **GitHub** | [nolanzhao/creative_tools](https://github.com/nolanzhao/creative_tools) |
| **技术栈** | Rust + Actix-web + PostgreSQL + Next.js 15 + TypeScript |
| **AI** | Gemini API (Nano Banana)、VEO API |

**dramora 可借鉴：**
- Rust 后端高性能处理的思路（如果未来有性能瓶颈）
- Gemini + VEO 的多模态 AI 组合

### 参考项目（非直接竞品）

| 项目 | 可借鉴点 |
|------|---------|
| **StoryDiffusion** (南开+字节) | Consistent Self-Attention 保证角色一致性，兼容 SD1.5/SDXL |
| **Index-AniSora** (B站) | RLHF 动画视频生成，千万级训练数据，动漫风格 |
| **AniMaker** (阿里+哈工大) | MCTS 驱动片段生成，节省 51% 计算资源 |
| **MovieAgent** (ShowLab) | 多 LLM Agent 模拟完整电影制作团队 |
| **AniME** (B站) | Director Agent + MCP 工具选择 + 7 个专业 Agent |
| **Komiko** | AI 漫画/韩漫/日漫/网漫工厂 |
| **AIMangaStudio** | React + Vite + Google GenAI 漫画创作 |

---

## Web 剪辑方案选型

### 候选方案对比

| 方案 | 定位 | 渲染方式 | 性能 | 适合场景 |
|------|------|---------|------|---------|
| **Remotion** (22k⭐) | 程序化视频生成框架 | 服务端 Chrome + FFmpeg | 近原生速度 | 自动化批量生成、参数化视频 |
| **OpenVideo** (~140⭐) | 交互式视频编辑器框架 | 客户端 WebCodecs + PixiJS | 硬件加速实时预览 | 拖拽式编辑 UI（CapCut/Canva 替代） |
| **ffmpeg.wasm** | 浏览器端 FFmpeg | 客户端 WASM (CPU only) | 比原生慢 10-12× | 格式转换、修剪、自定义管线 |
| **@cloudgpt/timeline-editor** | React 时间轴组件 | 纯前端 React | 取决于宿主 | 嵌入时间轴到现有页面 |
| **ClipJS** | 完整在线编辑器 | Next.js + Remotion + ffmpeg.wasm | 本地处理 | 开箱即用的 Web 剪辑工具 |
| **Seq** | AI 原生编辑器 | Next.js 16 + Canvas + FFmpeg WASM | 帧精确播放 | AI + 视频编辑一体化 |

### 推荐方案：全浏览器端渲染（OpenVideo + ffmpeg.wasm 混合管线）

**设计决策：零服务端渲染，全部在用户浏览器完成。**

**Why 全浏览器端：**
- 服务端 Chrome + FFmpeg 渲染大幅消耗 CPU/内存，多用户并发时成本不可控
- 用户素材已在浏览器中（COS 直传），没必要回传服务端再渲染
- 2025 年 WebCodecs + ffmpeg.wasm 混合管线已成熟，可覆盖 90%+ 剪辑场景
- 竞品 waoowaoo/Openframe 也都是客户端优先方案

**技术选型：**

| 任务 | 方案 | 性能 |
|------|------|------|
| **实时预览 + 交互编辑** | OpenVideo（WebCodecs + PixiJS） | 硬件加速，接近原生 |
| **MP4 导出渲染** | WebCodecs VideoEncoder + muxer | 硬件加速编码，接近原生速度 |
| **复杂格式转换/滤镜** | ffmpeg.wasm（放入 Web Worker） | CPU 软件编码，比原生慢 10-12×，仅兜底用 |
| **FCPXML 导出** | 纯前端生成 XML 文本 → 下载文件 | 瞬时完成（非渲染操作） |

**具体方案：**

```
TimelineExportPage (全浏览器端)
  ├─ 交互编辑层：OpenVideo (React 组件嵌入)
  │   ├─ 多轨道时间轴（视频/音频/字幕）
  │   ├─ 拖拽裁剪、转场特效
  │   ├─ WebCodecs 实时预览（硬件加速）
  │   └─ JSON 序列化项目状态 → 存入后端 / localforage
  │
  └─ 导出（全部在浏览器执行）
       ├─ MP4 导出：WebCodecs VideoEncoder 硬件加速 → 下载/上传 COS
       ├─ FCPXML 导出：纯前端生成 XML → 下载文件 → 可导入剪映/PR/FCPX/达芬奇
       └─ ffmpeg.wasm 兜底：仅用于 WebCodecs 不支持的格式（放入 Worker 不阻塞 UI）
```

**性能优化策略：**
- WebCodecs VideoEncoder 主线路（硬件加速，接近原生速度）
- ffmpeg.wasm 启用 Threads + SIMD（需 COOP/COEP 头，提速 1.5-3×）
- 大文件使用 OPFS（`navigator.storage.getDirectory()`）落盘，避免内存溢出
- 导出长视频时分段处理 + 最后拼接（`-c copy` remux）
- Web Worker 中执行所有渲染，不阻塞 UI 线程

**为什么不做服务端渲染：**
- 服务端 Chrome headless → 每并发用户 ~500MB+ 内存，GPU 资源昂贵
- 用户素材在 COS → 回传服务端渲染再上传 → 流量费用 + 延迟
- 浏览器端硬件加速编码已成熟，4K 视频导出在 M1/M2 Mac 上 < 实时速度
- 遇到极端情况（低端设备/超大文件）→ 可提示用户下载 FCPXML 用剪映/PR 导出

---

## 竞争定位 — dramora 的差异化优势

基于竞品分析，dramora 的核心差异化应聚焦：

| 能力 | waoowaoo | Openframe | **dramora（目标）** |
|------|----------|-----------|-------------------|
| 多 Agent 协作 | ❌ 固定 Worker | ❌ 单机工具 | **15 Agent DAG + Blackboard** |
| 人工审批流 | ❌ | ❌ | **ApprovalGate 体系** |
| 资产图谱 | 简单管理 | 角色/道具/场景 | **StoryMap + Asset Graph** |
| Web 剪辑 | 基础合成 | ❌ 无 | **全浏览器端剪辑** |
| 专业导出 | MP4 | FCPXML/EDL | **MP4 + FCPXML（桥接剪映/PR/FCPX/达芬奇）** |
| 多用户 SaaS | ✅ | ❌ | ✅ |
| 故事解析深度 | 基础提取 | 基础提取 | **多 Agent 深度分析** |
| 分镜工作台 | 自动分镜 | 自动分镜 | **人机协作分镜 + 提示词工程** |
| 人物关系图谱 | ❌ | ✅ | **StoryMap 可增强** |

---

## 更新后的 Phase 4 Web 剪辑详细设计

### 功能范围

```
TimelineExportPage 剪辑能力：
  ├─ 时间轴编辑
  │   ├─ 多轨道：视频轨、音频轨、字幕轨
  │   ├─ 片段拖拽排序
  │   ├─ 入点/出点裁剪（trim）
  │   └─ 网格吸附 + 缩放
  ├─ 转场特效
  │   ├─ 淡入淡出、叠化、推入推出
  │   └─ 转场时长可调
  ├─ 字幕编辑
  │   ├─ 时间轴对齐字幕
  │   ├─ 字幕样式（字体/大小/颜色/位置）
  │   └─ 从 Voice Agent 产出自动填充
  ├─ 预览
  │   ├─ WebCodecs 实时预览
  │   └─ 帧精确播放
  └─ 导出（全部在浏览器执行）
      ├─ MP4：WebCodecs VideoEncoder 硬件加速 → 下载 / 上传 COS
      ├─ FCPXML：纯前端生成 XML → 下载 .fcpxml 文件
      │   └─ 可导入：剪映专业版、CapCut Desktop、PR、FCPX、达芬奇
      └─ ffmpeg.wasm 兜底：仅用于 WebCodecs 不支持的格式（放入 Worker）
```

### 技术依赖

```json
{
  "前端（全浏览器端）": {
    "openvideo": "交互式时间轴编辑器核心（WebCodecs + PixiJS）",
    "@ffmpeg/ffmpeg": "格式转换兜底（放入 Web Worker + OPFS 落盘）",
    "@cloudgpt/timeline-editor": "备选时间轴组件（如果 OpenVideo 不够成熟）",
    "localforage": "浏览器端项目草稿持久化（IndexedDB）"
  },
  "后端": {
    "无需视频渲染服务": "全部在浏览器端完成",
    "仅需存储": "项目 JSON + 素材文件（复用现有 COS）"
  }
}
```

### 剪映/CapCut 兼容性

**关键发现：FCPXML 是通用桥接格式。**

| 剪辑软件 | 导入 FCPXML | 说明 |
|---------|------------|------|
| **剪映专业版** | ✅ 原生支持（V10.x+） | 设置 → 全局设置 → 开启"导入工程" |
| **CapCut Desktop** | ✅ 2025 版完整支持 | 支持 XML 导入/导出 |
| **Premiere Pro** | ✅ 原生支持 | 标准 FCP7 XML 交换格式 |
| **Final Cut Pro** | ✅ 原生格式 | FCPXML 是 FCPX 原生格式 |
| **达芬奇 Resolve** | ✅ 支持 | 标准 XML 导入 |

**dramora 导出策略：只做 FCPXML，覆盖全平台。**
- dramora 生成一份 FCPXML → 用户可导入任意主流剪辑软件
- 不需要单独适配每个软件的私有格式
- 剪映/CapCut 的私有格式（.jeproj / .caproj）是加密二进制，无公开文档，不应尝试逆向

### 与现有系统的集成点

- **Voice Agent 产出** → 自动填充字幕轨道（时间戳 + 文本）
- **Storyboard Shots** → 作为时间轴片段来源
- **Approval Gates** → 剪辑完成后触发 `final_timeline` 审批
- **Export Job** → 复用现有 `POST /episodes/{id}/exports` 流程

---

## 更新后的决策汇总

| # | 决策 | 选项 |
|---|------|------|
| 1 | 实施节奏 | 全四阶段规划，分批实施 |
| 2 | AI 端点 | **前端管理后台配置，存 DB 即时生效；4 能力独立端点 + 积分单价，无内置 Provider** |
| 3 | 编排模型 | 混合：DAG（生产流水线）+ Blackboard（审查层） |
| 4 | Phase 1 范围 | 基础设施 + 仅替换现有 5 Agent |
| 5 | Web 剪辑方案 | 全浏览器端：OpenVideo（交互编辑）+ WebCodecs（MP4导出）+ ffmpeg.wasm（兜底），零服务端渲染 |
| 6 | 专业导出 | FCPXML 单一桥接格式 → 可导入剪映/PR/FCPX/达芬奇/CapCut |
| 7 | 人物关系图谱 | Phase 2 增强 StoryMap 为交互式关系图谱 |
| 8 | 服务端渲染 | 不做。全浏览器端硬件加速编码，降低服务端成本 |
| 9 | 前端 UI 设计 | 详见 `research/ui-design-multi-agent.md`；AgentBoard + AgentPipeline(DAG) + AgentOutputPanel + BlackboardView |
| 10 | 角色一致性方案 | Character Bible（主描述锚点 + 色板 + LoRA）+ IP-Adapter FaceID Plus v2 集成 |
| 11 | 分镜提示词工程 | 3×3 Contact Sheet 结构 + "导演思维"镜头语言 + Character Anchor System |
| 12 | 迭代优化策略 | 引入 MAViS 3E 原则（Explore→Examine→Enhance）替代一次性生成 |
| 13 | 任务队列可靠性 | 参考 waoowaoo 的 DB↔BullMQ 三层对账 + Watchdog 巡检模式 |

---

---

# 第三轮调研：AI 漫剧深度优化

> 调研日期：2026-04-28
> 涵盖：waoowaoo 源码分析、角色一致性、提示词工程、新一代多 Agent 框架

## 一、waoowaoo 源码可借鉴的工程实践

### 1.1 SHADOW 计费模式

waoowaoo 的三态计费：OFF（不记账）/ SHADOW（记账不扣费，灰度观测）/ ENFORCE（正式扣费）。

**dramora 可借鉴**：Phase 3 Cost Controller Agent 上线前，先用 SHADOW 模式在真实流量下"旁听"成本数据，验证定价模型后再切 ENFORCE。

### 1.2 DB ↔ 任务队列三层对账

waoowaoo 防止"僵尸任务"的三层机制：

| 层级 | 机制 | dramora 当前状态 |
|------|------|-----------------|
| L1 创建即时校验 | Enqueue 时检查 dedupeKey 是否已有活跃任务 | 无，可直接加入 |
| L2 Watchdog 批量对账 | 每 60s 扫描 terminal/missing 状态并标记失败 | 无独立 Watchdog |
| L3 竞态保护 | terminal 给 90s 宽限期，missing 给 30s | 无 |

**dramora 可借鉴**：`generation_worker_service.go` 增加了 ProcessQueuedGenerationJobs 但缺少 zombie job 检测。Phase 1 可加入 L1 即时校验（成本低），Phase 2+ 加入 L2+L3。

### 1.3 Capability 组合定价

不按模型名定价，而是按 `模型 × 分辨率 × 时长 × 生成模式 × 是否带音频` 组合查询。

**dramora 可借鉴**：Phase 3 Cost Controller Agent 的定价目录应设计为组合矩阵而非简单模型名映射。预留 138KB JSON 定价文件结构。

---

## 二、角色一致性 — AI 漫剧 #1 技术难点

### 2.1 问题本质

> "上一秒黑发御姐，下一秒棕发萝莉" — 角色漂移是 AI 漫剧最大的质量杀手

### 2.2 三层解决方案（由浅入深）

| 层级 | 方案 | 成本 | 一致性 | dramora 集成时机 |
|------|------|------|--------|-----------------|
| **L1 提示词锚点** | Character Bible（主描述 + 色板 + 命名） | 零 | 60-70% | Phase 1 立即做 |
| **L2 IP-Adapter** | IP-Adapter FaceID Plus v2 (SDXL) | 低（ComfyUI 节点） | 85-90% | Phase 2 集成到 Seedance pipeline |
| **L3 LoRA 微调** | 20-50 张角色参考图 → Kohya_ss 训练 | 中（GPU 小时） | 95%+ | Phase 3 作为高级特性 |

### 2.3 Character Bible 结构（dramora 应扩展 StoryMap）

```
Character Bible (每个角色)
  ├─ 主描述锚点（只描述外观，不包含场景/动作）
  │   "maya，28岁，身材健美，黑色长发+蓝色挑染，
  │    标志性绿色眼睛，左眉上方小疤"
  ├─ 色彩档案（hex 色值）
  │   肤色 #E8C9A0  发色 #1A1A2E  挑染 #3B82F6
  │   眼睛 #22C55E  服装 #1F2937
  ├─ 多角度参考图（7 个标准角度）
  │   正面 / 背面 / 3/4左 / 3/4右 / 侧面左 / 侧面右 / T-pose
  ├─ 表情差分（≥6 个）
  │   中性 / 开心 / 愤怒 / 悲伤 / 惊讶 / 沉思
  ├─ 服装变体（按场景）
  │   scene_01: 黑色战斗服 + 护腕
  │   scene_02: 日常便装
  └─ LoRA 信息（Phase 3）
      trigger_word: "maya_char"
      lora_path: "loras/maya_v2.safetensors"
      recommended_scale: 0.75
```

**dramora 数据模型扩展**：`StoryMapItem` 增加 `character_bible` JSON 字段。前端 AssetsGraphPage 增加 "Character Bible" 编辑视图。

---

## 三、分镜提示词工程

### 3.1 核心转变：从文字思维到导演思维

| 文字思维（错误） | 导演思维（正确） |
|-----------------|-----------------|
| "主角走在雨夜的街道上，心情沉重" | "【中景 MCU】低角度仰拍，雨水沿主角脸颊滑落，霓虹灯光在湿漉漉的柏油路面形成紫色光晕，主角缓慢抬头凝视远方" |

### 3.2 3×3 Contact Sheet 分镜结构（推荐用于 dramora StoryboardAgent）

```
第一行：环境建立
  格1 大远景(ELS)  — 交代世界与情绪基调
  格2 全景(LS)    — 角色完整全身 + 环境关系
  格3 中远景(MLS)  — 膝盖以上，开始体现人物状态

第二行：情绪叙事
  格4 中景(MS)    — 腰部以上，明确动作/姿态
  格5 中特写(MCU) — 胸部以上，表情清晰
  格6 特写(CU)    — 面部特写，眼神是重点

第三行：强化电影感
  格7 大特写(ECU) — 细节（眼睛/手/道具），象征意义
  格8 低角度(LA)  — 仰视，力量/决心/英雄感
  格9 高角度(HA)  — 俯视，孤独/压力/空间关系
```

### 3.3 dramora PromptPackAgent 提示词模板升级

现有 deterministic prompt assembly → Phase 2 升级为：

```
ShotPromptPack 结构扩展:
  ├─ shot_id
  ├─ shot_code
  ├─ scene_context        (现有)
  ├─ camera_spec:         ← 新增：3×3 Contact Sheet 镜头指令
  │   ├─ shot_size         // ELS/LS/MLS/MS/MCU/CU/ECU
  │   ├─ camera_angle      // eye-level/low-angle/high-angle/dutch
  │   ├─ camera_movement   // static/push-in/pull-out/pan/tilt/tracking
  │   └─ composition       // rule-of-thirds/center-frame/leading-lines
  ├─ character_anchors[]:  ← 新增：锁定角色引用
  │   └─ { character_id, outfit, expression, pose }
  ├─ lighting_spec:        ← 新增：光效指令
  │   ├─ key_light         // 主光源方向+色温
  │   ├─ fill_light        // 补光
  │   └─ atmosphere        // 雾气/粒子/体积光
  └─ consistency_tags[]:   ← 新增：一致性约束标签
      // "preserve_maya_hair_color", "preserve_scene_01_lighting"
```

---

## 四、新一代多 Agent 框架的启示

### 4.1 MAViS 3E 原则（EACL 2026）

MAViS (Virginia Tech / Eyeline Studios) 提出 **Explore→Examine→Enhance** 迭代循环：

```
一次生成                      →  MAViS 3E 循环
─────────────────────────────────────────────────
Agent 生成 → 即用              Explore: 生成候选
                                Examine: 审查质量/完整性
                                Enhance: 迭代精炼
                                     ↓
                               不满意 → 回到 Explore
                               满意 → 输出
```

**dramora 应用**：Phase 2 的 StoryboardAgent 和 PromptPackAgent 不应一次性生成。应生成 ≥2 个候选，由 Director Agent 或人工审批选择最佳。

### 4.2 AniMaker MCTS 驱动生成（SIGGRAPH Asia 2025）

AniMaker 用蒙特卡洛树搜索（MCTS）生成多个候选片段后择优：

```
传统: 1 shot → 1 次生成 → 50% 满意率
MCTS: 1 shot → 3 候选 → 评估 → 最优 → 85% 满意率
      计算量: 3× 但节省了重做时间 → 净省 51%
```

**dramora 应用**：Phase 3 的视频生成可引入 MCTS 策略。不是每个 shot 只生成一个视频，而是生成 2-3 个候选，由 Reviewer Agent 自动评分选最优。

### 4.3 BookAgent 全局一致性修复（ACL 2026）

BookAgent 的核心创新：不是"每个页面独立生成然后拼接"，而是 **跨页面全局角色身份一致性检测 + 自动修复**。

**dramora 应用**：Phase 2 Continuity Supervisor Agent 的设计应参考 BookAgent —— 不仅检测问题，还要**自动建议修复方案**（"C01 角色在第 5 镜中发色漂移，建议用 IP-Adapter 重新生成该镜头"）。

---

## 五、对四阶段路线的细化建议

### Phase 1 补充项

| 补充 | 说明 | 优先级 |
|------|------|--------|
| Character Bible 数据模型 | `StoryMapItem` 增加 `character_bible` JSON 字段 | 高 |
| 任务 Enqueue 即时校验 | 创建 Job 时检查 dedupeKey 防重复 | 高 |
| 分镜提示词结构扩展 | `ShotPromptPack` 增加 `camera_spec` + `character_anchors` | 中 |

### Phase 2 补充项

| 补充 | 说明 | 优先级 |
|------|------|--------|
| MAViS 3E 循环 | StoryboardAgent/PromptPackAgent 生成 ≥2 候选 | 高 |
| 3×3 Contact Sheet | StoryboardAgent 按 9 格结构出分镜 | 高 |
| IP-Adapter 集成 | Seedance pipeline 增加 FaceID 参考图注入 | 中 |

### Phase 3 补充项

| 补充 | 说明 |
|------|------|
| MCTS 视频候选生成 | AniMaker 模式，每个 shot 生成 2-3 候选，自动评优 |
| LoRA 训练集成 | 用户上传 20+ 角色参考图 → 自动触发 LoRA 训练 → 注入 Seedance |
| Continuity 自动修复 | 参考 BookAgent 全局一致性检测 + 修复建议 |
| SHADOW 计费模式 | Cost Controller 先用 SHADOW 灰度，验证定价后切 ENFORCE |

### Phase 4 补充项

| 补充 | 说明 |
|------|------|
| Capability 组合定价矩阵 | 模型 × 分辨率 × 时长 × 模式 × 音频 组合定价目录 |
| DB↔Job 对账机制 | L2 批量对账 + L3 竞态保护（参考 waoowaoo） |
| Character Bible 编辑器 | AssetsGraphPage 增加可视化 Character Bible 编辑 |

---

---

# 第四轮：积分系统 + 遗漏领域规划

> 调研日期：2026-04-28

## 一、积分/计费系统设计

### 1.1 核心设计原则

基于 2025-2026 年 AI SaaS 计费最佳实践（Lago、Flexprice、Autumn、waoowaoo）：

| 原则 | 说明 | 反模式 |
|------|------|--------|
| **按产出计费，不按用量** | "每个镜头生成"而非"每 token" | 按 API 调用次数计费 |
| **积分抽象层** | 用户看到积分，后端映射到实际成本 | 直接暴露 API 价格 |
| **免费激活 → 付费转化** | 新用户送免费积分体验完整流程 | 一上来就要付费 |
| **组合定价** | 模型×分辨率×时长×模式×音频 | 模型名统一定价 |
| **SHADOW 灰度** | 先记账不扣费，验证定价模型 | 拍脑袋定价直接上线 |

### 1.2 积分系统架构

```
┌─────────────────────────────────────────────────────────┐
│                    积分系统架构                            │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────┐    ┌──────────┐    ┌──────────────────┐   │
│  │ 用户钱包  │    │ 定价目录  │    │ 交易流水账        │   │
│  │          │    │          │    │                  │   │
│  │ balance  │    │ 模型     │    │ id              │   │
│  │ frozen   │    │ 分辨率   │    │ user_id         │   │
│  │ expires  │    │ 时长     │    │ amount (积分)    │   │
│  │          │    │ 模式     │    │ type (消费/充值)  │   │
│  └────┬─────┘    │ 音频     │    │ capability      │   │
│       │          │ → 积分   │    │ project_id      │   │
│       │          └────┬─────┘    │ episode_id       │   │
│       │               │          │ shot_id          │   │
│       │    ┌──────────┴──────┐   │ model_name      │   │
│       │    │  Capability     │   │ actual_cost(¥)  │   │
│       └───→│  Pricing Engine │──→│ created_at      │   │
│            └─────────────────┘   └──────────────────┘   │
│                                                          │
│  BILLING_MODE: OFF | SHADOW | ENFORCE                    │
│  OFF     = 不计费（开发/自部署）                           │
│  SHADOW  = 记账不扣费（灰度验证定价）                       │
│  ENFORCE = 正式扣费                                       │
└─────────────────────────────────────────────────────────┘
```

### 1.3 积分定价（管理员环境变量配置）

> 无内置定价文件。积分单价全部由管理员通过环境变量设定，与端点配置一一对应。

```
积分单价 = 端点配置中的 *_CREDITS_* 环境变量:

  PROVIDER_CHAT_CREDITS_PER_CALL=5       → 每次 LLM 调用消耗 5 积分
  PROVIDER_IMAGE_CREDITS_PER_CALL=10     → 每次生图消耗 10 积分
  PROVIDER_VIDEO_CREDITS_PER_SECOND=20   → 每秒视频消耗 20 积分
  PROVIDER_AUDIO_CREDITS_PER_CHAR=1      → 每字 TTS 消耗 1 积分
```

**典型一集漫剧积分消耗参考**:

### 1.4 积分消耗预估（典型一集漫剧）

| 阶段 | 操作 | 积分 |
|------|------|------|
| 故事解析 | 5 Agent LLM 分析 | 25 |
| 角色设计 | 4 角色 × 多角度参考图 | 80 |
| 场景设计 | 4 场景 × 概念图 | 60 |
| 道具设计 | 2 道具 × 参考图 | 20 |
| 分镜规划 | LLM 分镜 + 提示词工程 | 30 |
| 视频生成 | 12 镜头 × 5s 720p | 1,200 |
| TTS 配音 | 3 角色 × 30s | 45 |
| 导出 | 1080p 最终视频 | 20 |
| **合计** | | **~1,480 积分/集** |

### 1.5 积分套餐设计

| 套餐 | 积分 | 价格 | 定位 |
|------|------|------|------|
| 免费体验 | 200 | ¥0 | 完成一次故事解析 + 3 个镜头 |
| 创作者 | 2,000/月 | ¥29/月 | 制作 1 集/月 |
| 专业版 | 5,000/月 | ¥69/月 | 制作 3 集/月 |
| 工作室 | 20,000/月 | ¥199/月 | 制作 10+ 集/月 |
| 企业定制 | 按需 | 按需 | API 接入 + 私有化部署 |

### 1.6 数据模型扩展

```go
// internal/domain/credits.go (新增)

type CreditWallet struct {
    ID        string
    UserID    string
    Balance   int       // 可用积分
    Frozen    int       // 冻结中（正在执行的任务）
    ExpiresAt *time.Time // 积分过期时间
}

type CreditTransaction struct {
    ID           string
    UserID       string
    Amount       int        // 正=充值, 负=消费
    Type         string     // "charge" | "consume" | "refund" | "expire"
    Capability   string     // story_analysis | video_generation | ...
    ProjectID    *string
    EpisodeID    *string
    ShotID       *string
    ModelName    string
    ActualCost   float64    // 实际 API 成本（¥）
    CreditCost   int        // 消耗积分
    Metadata     JSON
    CreatedAt    time.Time
}

type CapabilityPrice struct {
    Capability       string
    Model            string
    Resolution       string  // 可选
    DurationSeconds  int     // 可选
    Mode             string  // 可选
    Audio            bool    // 可选
    Credits          int
    EstimatedCostCNY float64
}

// BillingMode 环境变量控制
// OFF:     不检查积分，不记录流水（开发/自部署）
// SHADOW:  不扣积分，但记录流水和预估成本（灰度验证定价）
// ENFORCE: 检查积分余额 → 扣积分 → 记录流水（正式运营）
```

### 1.7 积分检查流程

```
POST /episodes/{id}/story-analysis/start
  │
  ├─ BILLING_MODE=OFF → 直接执行
  │
  ├─ BILLING_MODE=SHADOW
  │   ├─ 查询 CapabilityPrice (story_analysis + deepseek-chat = 5 credits)
  │   ├─ 记录 CreditTransaction (amount=0, credit_cost=5, type="shadow")
  │   └─ 执行任务
  │
  └─ BILLING_MODE=ENFORCE
      ├─ 查询 CapabilityPrice
      ├─ 检查 Wallet.Balance >= 5
      │   ├─ 不足 → 返回 402 Payment Required
      │   └─ 充足 → Wallet.Balance -= 5, Wallet.Frozen += 5
      ├─ 记录 CreditTransaction (amount=-5, type="consume")
      ├─ 执行任务
      └─ 任务完成 → Wallet.Frozen -= 5
          任务失败 → Wallet.Balance += 5, Wallet.Frozen -= 5 (退款)
```

---

## 二、遗漏领域检查

### 2.1 全面规划检查清单

| # | 领域 | 状态 | 说明 |
|---|------|------|------|
| 1 | 多 Agent 协作 | ✅ 已规划 | 四阶段路线 |
| 2 | Web 剪辑 | ✅ 已规划 | Phase 4 全浏览器端 |
| 3 | 前端 UI 设计 | ✅ 已规划 | AgentBoard + DAG + OutputPanel |
| 4 | 角色一致性 | ✅ 已规划 | L1-L3 三层方案 |
| 5 | 分镜提示词工程 | ✅ 已规划 | 3×3 Contact Sheet |
| 6 | 积分/计费系统 | ✅ 本次补充 | SHADOW + Capability 定价 |
| 7 | 用户认证与权限 | ⚠️ 未规划 | 多用户、角色权限、团队协作 |
| 8 | 内容安全与合规 | ⚠️ 未规划 | 敏感内容过滤、版权检测 |
| 9 | API 开放平台 | ⚠️ 未规划 | 第三方接入、Webhook |
| 10 | 国际化 i18n | ⚠️ 未规划 | 目前仅中文，是否需要英文/日文 |
| 11 | 数据分析看板 | ⚠️ 未规划 | 创作数据统计、用量报表 |
| 12 | 移动端适配 | ⚠️ 未规划 | 是否需要移动端 |

### 2.2 遗漏领域简要设计

#### 用户认证与权限（完整设计）

**当前状态**: 无用户系统，前端无登录。

**Phase 2 引入完整体系**：

### 两层权限模型

```
系统级角色 (全局):
  admin  — 超级管理员，管理后台全部权限
  user   — 普通用户

团队空间 (Workspace, Phase 2+):
  系统级 role 决定能不能进管理后台
  Workspace 决定能不能协作编辑项目

  两层独立，互不干扰:
    · admin 在 workspace 里可能只是 member（不管那个团队的项目）
    · user 在自己的 workspace 里是 owner（完全控制自己的团队）
```

### 数据模型

```sql
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    avatar_url TEXT DEFAULT '',
    role TEXT NOT NULL DEFAULT 'user',        -- admin | user (系统级)
    is_active BOOLEAN DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Workspace = 团队空间
CREATE TABLE workspaces (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_by TEXT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE workspace_members (
    workspace_id TEXT NOT NULL REFERENCES workspaces(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    role TEXT NOT NULL DEFAULT 'member',       -- owner | admin | member | viewer
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (workspace_id, user_id)
);

CREATE TABLE user_api_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    name TEXT NOT NULL DEFAULT '',
    key_prefix TEXT NOT NULL,
    key_hash TEXT NOT NULL,
    last_used_at TIMESTAMP,
    expires_at TIMESTAMP,
    is_active BOOLEAN DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 系统级权限 (admin vs user)

| 操作 | admin | user |
|------|-------|------|
| 管理端点配置 | ✅ | ❌ |
| 管理用户（列表/禁用/改角色） | ✅ | ❌ |
| 积分套餐 + 手动充值 + 兑换码 | ✅ | ❌ |
| 查看全局积分流水 | ✅ | ❌ |
| 创建项目 | ✅ | ✅ |
| 创建/编辑剧集 | ✅ | ✅ |
| 触发解析/生成 | ✅ | ✅ |
| 编辑分镜/提示词 | ✅ | ✅ |
| 剪辑/导出 | ✅ | ✅ |
| 兑换积分码 | ✅ | ✅ |
| 修改自己的账号设置 | ✅ | ✅ |

### Workspace 内权限 (团队协作)

| 操作 | owner | admin | member | viewer |
|------|-------|-------|--------|--------|
| 管理 workspace（改名/删队） | ✅ | ❌ | ❌ | ❌ |
| 邀请/移除成员 | ✅ | ✅ | ❌ | ❌ |
| 改成员角色 | ✅ | ✅ | ❌ | ❌ |
| 编辑所有项目 | ✅ | ✅ | ❌ | ❌ |
| 编辑自己创建的项目 | ✅ | ✅ | ✅ | ❌ |
| 查看所有项目 | ✅ | ✅ | ✅ | ✅ |
| 创建新项目 | ✅ | ✅ | ✅ | ❌ |
| 导出 | ✅ | ✅ | ✅ | ✅ |

### 单用户无感

```
Phase 1: 无登录 → 所有操作无需 workspace
Phase 2: 引入登录 → 用户注册时自动创建默认 private workspace
         → 用户感知不到 workspace 存在
         → 仅在"邀请成员"时 workspace 概念才显现
```

### 前端页面

```
AccountSettingsPage (/account/settings)   ← 个人信息 + 密码 + API Key
CreditsPage (/account/credits)            ← 积分余额 + 兑换码 + 流水
WorkspacePage (/workspace)                ← 成员列表 + 邀请 + 角色管理
AdminCreditsPage (/admin/credits)         ← 充值 + 兑换码 + 流水 + 套餐
AdminUsersPage (/admin/users)             ← 用户管理
AdminSettingsPage (/admin/settings)       ← 端点配置
LoginPage (/login) / RegisterPage
```

---

## 兑换码系统（核心商业模式）

> dramora 靠卖积分盈利。管理员在后台管理充值 + 批量生成兑换码 → 分发给用户 → 用户在前端兑换。

### 兑换码格式

```
DRAM-A1B2-C3D4-E5F6
  前缀  随机串  校验位
  4位   12位    2位(CRC16)
```

### 数据模型

```sql
CREATE TABLE redemption_batches (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,                    -- 批次名称 "抖音推广活动"
    credit_value INT NOT NULL,             -- 每个码积分数
    max_uses_per_code INT DEFAULT 1,       -- 1=一次性
    total_codes INT NOT NULL,
    used_count INT DEFAULT 0,
    expires_at TIMESTAMP,
    created_by TEXT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE redemption_codes (
    id TEXT PRIMARY KEY,
    batch_id TEXT NOT NULL REFERENCES redemption_batches(id),
    code TEXT NOT NULL UNIQUE,             -- 唯一索引
    credit_value INT NOT NULL,
    max_uses INT DEFAULT 1,
    used_count INT DEFAULT 0,
    is_active BOOLEAN DEFAULT 1,
    expires_at TIMESTAMP
);

CREATE TABLE redemption_records (
    id TEXT PRIMARY KEY,
    code_id TEXT NOT NULL REFERENCES redemption_codes(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    credits_awarded INT NOT NULL,
    redeemed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 核销流程

```
用户输入 DRAM-A1B2-C3D4-E5F6
  → 查 code 表（唯一索引）
  → 校验: 未过期 + 未禁用 + used_count < max_uses
  → 乐观锁 UPDATE codes SET used_count+1 WHERE used_count < max_uses
  → INSERT redemption_record + credit_transaction(type="redeem")
  → UPDATE wallet balance + credits
  → 返回 { credits_awarded: 500, new_balance: 1500 }
```

### API

```
POST /api/v1/credits/redeem               → 用户兑换
POST /api/v1/admin/redemption/batches      → 批量生成码
GET  /api/v1/admin/redemption/batches      → 批次列表
GET  /api/v1/admin/redemption/batches/{id} → 批次详情 + 核销明细
POST /api/v1/admin/redemption/batches/{id}:disable → 禁用批次
GET  /api/v1/admin/redemption/csv/{id}     → 导出 CSV
POST /api/v1/admin/credits/recharge        → 手动给用户充值
GET  /api/v1/admin/credits/transactions    → 积分流水
```

### 管理后台 (AdminCreditsPage)

```
├─ 手动充值 Tab: 选用户 → 填积分数 → 确认
├─ 兑换码 Tab: 生成批次 → 查看统计 → 导出CSV → 禁用
├─ 积分流水 Tab: 筛选 + 流水表
└─ 积分套餐 Tab: 创建/编辑/停用套餐
```

### 权限中间件

```go
// internal/httpapi/middleware/auth.go

func RequireAuth(userRepo UserRepository) func(http.Handler) http.Handler {
    // 从 Authorization: Bearer <token> 解析 JWT
    // 验证 token → 查询 user → 注入 ctx
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
    // 从 ctx 取 user → 检查 role 是否在允许列表中
    // 不满足 → 403
}

// 路由示例:
r.Group(func(r chi.Router) {
    r.Use(RequireAuth(userRepo))
    r.Get("/api/v1/projects", api.listProjects)           // 所有角色
    r.Post("/api/v1/projects", RequireRole("admin","creator")) // creator+
})
r.Group(func(r chi.Router) {
    r.Use(RequireAuth(userRepo), RequireRole("admin"))
    r.Get("/api/v1/admin/providers", api.listProviders)   // admin only
})
```

### API 路由

```
认证 (无需登录):
  POST /api/v1/auth/register          → 注册
  POST /api/v1/auth/login             → 登录，返回 JWT
  POST /api/v1/auth/refresh           → 刷新 token

账号设置 (需登录):
  GET  /api/v1/account/profile        → 查看个人信息
  POST /api/v1/account/profile:update → 修改个人信息
  POST /api/v1/account/password:change → 修改密码
  GET  /api/v1/account/api-keys       → 查看 API Key 列表
  POST /api/v1/account/api-keys:create → 创建 API Key
  POST /api/v1/account/api-keys/{id}:revoke → 吊销 API Key

用户管理 (仅 admin):
  GET  /api/v1/admin/users            → 用户列表（分页+搜索）
  POST /api/v1/admin/users/{id}:disable → 禁用用户
  POST /api/v1/admin/users/{id}:role   → 修改角色
```

### 前端页面

```
AccountSettingsPage (/account/settings)     ← 用户自己的账号设置
  ├─ 个人信息 Tab: 名称 / 邮箱 / 头像
  ├─ 安全 Tab: 修改密码
  └─ API Key Tab: 创建/查看/吊销 API Key

AdminUsersPage (/admin/users)              ← 管理员用户管理
  ├─ 用户列表（搜索/分页）
  └─ 操作：禁用/启用/修改角色

LoginPage (/login)                          ← 登录
RegisterPage (/register)                    ← 注册
```

#### 内容安全

**当前状态**: 无内容审核。

**建议**: Phase 3 Safety Agent 负责 + 接入第三方审核 API：

```
内容安全检查点:
  1. 故事源输入    → 敏感词过滤（政治/暴恐/色情）
  2. LLM 产出      → Safety Agent 审查（已在 Phase 3 规划）
  3. 生成图片/视频  → 接入阿里云/腾讯云内容安全 API
  4. 导出内容       → 最终审查 + 水印

敏感词库:
  - 内置基础词库（政治/暴恐/色情）
  - 支持自定义白名单/黑名单
```

#### 数据分析

**建议**: Phase 4 增加创作者数据看板：

```
指标:
  - 项目数 / 剧集数 / 镜头数
  - Agent 调用次数 / 成功率
  - 积分消耗趋势
  - 视频生成量 / 导出量
  - 平均每集耗时 / 成本

实现:
  - HomePage 增加 Analytics 卡片
  - 复用现有 dashboard-grid + hero-stat-card 模式
```

---

## 三、更新后的决策汇总

| # | 决策 | 选项 |
|---|------|------|
| 1 | 实施节奏 | 全四阶段规划，分批实施 |
| 2 | AI 端点 | **前端管理后台配置，存 DB 即时生效；4 能力独立端点 + 积分单价，无内置 Provider** |
| 3 | 编排模型 | 混合：DAG + Blackboard |
| 4 | Phase 1 范围 | 基础设施 + 替换现有 5 Agent |
| 5 | Web 剪辑方案 | 全浏览器端：OpenVideo + WebCodecs + ffmpeg.wasm |
| 6 | 专业导出 | FCPXML 单一桥接 → 剪映/PR/FCPX/达芬奇/CapCut |
| 7 | 服务端渲染 | 不做，全浏览器端硬件加速 |
| 8 | 前端 UI 设计 | AgentBoard + DAG + OutputPanel + BlackboardView |
| 9 | 角色一致性 | Character Bible (L1) → IP-Adapter (L2) → LoRA (L3) |
| 10 | 分镜提示词 | 3×3 Contact Sheet + 导演思维 + Character Anchor |
| 11 | 迭代优化 | MAViS 3E — Explore→Examine→Enhance |
| 12 | 任务可靠性 | DB↔Job 三层对账 + Watchdog |
| 13 | **积分计费**：**Capability 组合定价 + SHADOW→ENFORCE 灰度 + 按产出计费 + 兑换码系统 + 管理后台充值** |
| 14 | **用户与权限**：**系统级 admin/user + Workspace 内 owner/admin/member/viewer + 对象级权限 + 单用户无感** |
| 15 | **内容安全** | **Phase 3 Safety Agent + 第三方审核 API** |
| 16 | **数据分析** | **Phase 4 创作者数据看板** |
| 17 | **API 约束** | **仅 GET/POST，写操作用 POST + :action 后缀，禁止 PUT/PATCH/DELETE** |
| 18 | **自定义端点** | **前端管理后台配置，存 DB 即时生效；4 能力独立端点 + 积分单价，无内置 Provider** |
| 19 | **电商短视频** | **独立流水线模块；商品图+描述→LLM营销脚本→分镜→视频+配音+字幕+多平台尺寸** |
| 20 | **抽卡机制** | **生成 3-5 个候选 → 用户挑选最佳 → 不满可重抽（扣积分）；未选中候选保留可回溯** |
| 21 | **素材库** | **GalleryPage 统一管理；按项目/剧集/镜头分组；支持预览/对比/锁定/删除/导出** |

---

---

# 第五轮：电商商品短视频模块

> 调研日期：2026-04-28
> 参考项目：MoneyPrinterTurbo (56K⭐)、AnchorCrafter、SkyReels-V3、Shopee MUG-V 10B、AdGen、daihuo-videoforge

## 一、漫剧 vs 电商 — 两条流水线对比

dramora 当前设计偏向"长内容叙事"（漫剧），电商短视频是完全不同的产品形态：

| 维度 | 漫剧流水线 | 电商短视频流水线 |
|------|-----------|----------------|
| **输入** | 小说/故事文本（数千字） | 商品图片 + 描述（几百字） |
| **产出** | 多集剧集（每集 1-3 分钟） | 单个广告短视频（15-60 秒） |
| **Agent 数量** | 15 个 | 4-5 个 |
| **分镜复杂度** | 高（9 格 Contact Sheet） | 低（3-5 镜头） |
| **核心难点** | 角色一致性、叙事连贯 | 商品质感、卖点突出、转化率 |
| **导出格式** | 16:9 横屏/竖屏 | 9:16 竖屏（TikTok/抖音/小红书） |
| **制作周期** | 数小时/集 | 数分钟/视频 |

## 二、电商短视频 Agent 流水线

```
ProductSource ──→ MarketingAgent ──→ VisualAgent ──→ VideoAgent ──→ ExportAgent
 (商品输入)       (营销脚本)         (视觉分镜)       (视频生成)       (导出发布)
```

### 2.1 各 Agent 职责

| Agent | 输入 | 输出 | 参考方案 |
|-------|------|------|---------|
| **MarketingAgent** | 商品名称、描述、卖点、价格 | 营销脚本（Hook + 卖点 + CTA）、口播文案 | MoneyPrinterTurbo LLM 脚本 |
| **VisualAgent** | 商品图 + 脚本 | 3-5 镜头分镜（每镜描述 + 时长 + 字幕） | AdGen GPT-4 分镜 |
| **VideoAgent** | 分镜 + 商品图 | 每个镜头的视频片段 | Wan2.2-I2V / SkyReels-V3 / MUG-V |
| **ExportAgent** | 视频片段 + 配音 + 字幕 + BGM | 最终短视频（多尺寸） | FFmpeg 合成 |

### 2.2 MarketingAgent Prompt 模板

```
你是一个电商营销文案专家。请为以下商品生成 15-30 秒短视频脚本。

商品名称：{product_name}
商品描述：{product_description}
核心卖点：{selling_points}
目标平台：{platform}  (TikTok/抖音/小红书/快手)

要求：
1. 前 3 秒必须有强力 Hook（问题/惊讶/好奇心）
2. 中间展示 2-3 个核心卖点
3. 结尾清晰 CTA（购买/点击/关注）
4. 口语化、有情绪张力

输出 JSON：
{
  "hook": "前3秒钩子文案",
  "scenes": [
    {"duration": 5, "visual": "画面描述", "narration": "口播文案", "subtitle": "字幕文字"},
    ...
  ],
  "cta": "结尾行动号召"
}
```

### 2.3 与漫剧流水线的关系

```
dramora Studio
  ├─ 漫剧模式 (现有)
  │   ├─ StoryAnalysisPage → 故事解析 + 15 Agent
  │   ├─ StoryboardPage   → 分镜工作台
  │   └─ TimelineExportPage → 时间线 + 导出
  │
  └─ 电商模式 (新增)
      ├─ ShortVideoPage    → 商品输入 + 脚本生成 + 视频预览
      └─ 复用 StoryboardPage 分镜能力（简化版）
      └─ 复用 TimelineExportPage 导出能力
```

**复用现有能力**：
- 端点配置（chat/image/video/audio）完全相同
- WebCodecs 浏览器端导出完全复用
- 积分计费完全复用
- FCPXML 导出支持剪映二次编辑

## 三、ShortVideoPage 页面设计

```
ShortVideoPage (/short-video)
  ├─ 商品输入区
  │   ├─ 商品图片上传（主图 + 2-4 张细节图）
  │   ├─ 商品名称 / 描述 / 卖点 / 价格
  │   └─ 目标平台选择（抖音 / TikTok / 小红书 / 快手 / 通用）
  │
  ├─ 脚本预览区
  │   ├─ Hook 文案展示
  │   ├─ 分镜卡片（3-5 镜）
  │   └─ [重新生成脚本]
  │
  ├─ 视频预览区
  │   ├─ 竖屏播放器（9:16）
  │   ├─ 字幕叠加预览
  │   └─ 配音试听
  │
  └─ 导出区
      ├─ 尺寸选择（1080×1920 竖屏 / 其他）
      ├─ 格式选择（MP4 / 直接发布）
      └─ [导出视频] [导出 FCPXML 到剪映]
```

## 四、关键参考项目的可借鉴点

| 项目 | 可借鉴点 | dramora 应用 |
|------|---------|-------------|
| **MoneyPrinterTurbo** | LLM 文案→素材匹配→TTS→字幕→BGM 全自动流水线 | MarketingAgent 脚本生成逻辑 |
| **AnchorCrafter** | 人-物自然交互视频生成 | 未来可集成数字人带货 |
| **SkyReels-V3** | 多张参考图→视频，一致性高 | VideoAgent 商品图→视频 |
| **Shopee MUG-V 10B** | 电商场景优化的视频模型 | 如果管理员配置了 MUG-V 端点 |
| **AdGen** | 商品 URL→爬取→脚本→视频 | 商品输入方式参考 |
| **daihuo-videoforge** | 多平台尺寸适配、品牌风格定制 | ExportAgent 多尺寸导出 |

## 五、实现优先级

电商短视频是**独立模块**，与漫剧流水线共享基础设施，但不阻塞漫剧开发：

```
Phase 1-2: 漫剧基础设施 (现阶段)
Phase 2.5: 电商短视频 MVP
  ├─ ShortVideoPage 新页面
  ├─ MarketingAgent (LLM 脚本)
  ├─ VisualAgent (分镜)
  ├─ 复用 VideoAgent + ExportAgent
  └─ 3-5 镜头短视频输出

Phase 3+: 电商增强
  ├─ 数字人带货 (AnchorCrafter / SkyReels-V3 集成)
  ├─ 批量商品视频生成
  ├─ 多平台一键发布
  └─ A/B 测试不同脚本版本
```

## 六、电商积分消耗参考

| 操作 | 积分 |
|------|------|
| 生成营销脚本 (LLM) | 5 |
| 商品图→视频 (每镜头 5s) | 50 × 镜数 |
| TTS 配音 (30s) | 15 |
| 导出 (1080×1920) | 10 |
| **合计（3 镜头）** | **~180 积分/视频** |
| **合计（5 镜头）** | **~280 积分/视频** |

对比漫剧 ~1,480 积分/集，电商短视频成本约为 1/5 到 1/8。

---

---

# 第六轮：抽卡机制 + 素材库

> AI 视频生成可用率仅 15-20%，行业核心痛点。一次生成多候选 + 用户挑选 = "抽卡"。

## 一、抽卡机制设计

### 1.1 行业背景

- AI 视频生成的原始可用率仅 **15-20%**（中银证券 2025 研报）
- 创作者需大量生成后人工筛选 — 行业称为「抽卡」
- 催生了新职业 **「抽卡师」**（AI 提示词工程师）
- 字节 Seedance 2.0 将可用率拉到 90%，但仍需从多候选中择优

### 1.2 dramora 抽卡流程

```
用户触发"生成视频"
  │
  ├─ 扣除积分（N × 候选数）
  │   例：100 积分/候选 × 3 候选 = 300 积分
  │
  ├─ 并行生成 3-5 个候选
  │   ├─ 候选 #1 (seed=42)  → 完成
  │   ├─ 候选 #2 (seed=99)  → 完成
  │   └─ 候选 #3 (seed=17)  → 完成
  │
  ├─ 展示候选列表（GalleryView 抽卡界面）
  │   ┌──────────┬──────────┬──────────┐
  │   │ 候选 #1   │ 候选 #2   │ 候选 #3   │
  │   │ [预览]    │ [预览]    │ [预览]    │
  │   │ ⭐⭐⭐    │ ⭐⭐⭐⭐  │ ⭐⭐      │
  │   │ [选用]    │ [选用]    │ [选用]    │
  │   └──────────┴──────────┴──────────┘
  │
  └─ 用户操作:
      ├─ [选用] → 锁定该候选为正式版本
      ├─ [重抽] → 再生成 3 个候选（再扣 300 积分）
      └─ [保留] → 多个候选保留在素材库，后续可回溯
```

### 1.3 抽卡界面设计

```
ShotCard 内嵌抽卡模式:
  ┌─ ShotCard (镜头 #3) ──────────────────────────────┐
  │                                                      │
  │  当前版本: 候选 #2 (选用)                             │
  │                                                      │
  │  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐               │
  │  │ 候1   │ │ 候2 ✓│ │ 候3   │ │ ➕    │               │
  │  │ 5s    │ │ 5s   │ │ 4.8s │ │ 再抽  │               │
  │  │ ⭐⭐  │ │ ⭐⭐⭐⭐│ │ ⭐⭐⭐ │ │300积分│              │
  │  │ [预览]│ │ [选用]│ │ [预览]│ │      │               │
  │  └──────┘ └──────┘ └──────┘ └──────┘               │
  │                                                      │
  │  [全屏对比模式] [全部保留到素材库]                      │
  └──────────────────────────────────────────────────────┘
```

**候选状态**:
| 状态 | 标记 | 说明 |
|------|------|------|
| generating | 旋转加载 + 进度 | 生成中 |
| ready | 缩略图 + 时长 | 可选用的候选 |
| selected | ✅ + 紫色边框 | 当前选用 |
| rejected | 灰色蒙层 | 已放弃（但保留可回溯） |

---

## 二、素材库（视频资产管理）

### 2.1 存储方案

```
存储层级:
  项目 (Project)
    └─ 剧集 (Episode)
         └─ 镜头 (Shot)
              └─ 候选视频 (Candidate[])

数据库 (SQLite/PostgreSQL):
  assets 表 (已有, 扩展):
    id, episode_id, shot_id, candidate_index, kind,
    url, thumbnail_url, duration_ms, resolution,
    file_size_bytes, seed, prompt_pack_id,
    status, is_selected, created_at

文件存储:
  开发环境: 本地文件 data/media/{asset_id}.mp4
  生产环境: COS/S3  {bucket}/media/{asset_id}.mp4
```

### 2.2 新增 GalleryPage（素材库页面）

```
GalleryPage (/gallery)
  ├─ 筛选栏
  │   ├─ 按项目/剧集筛选
  │   ├─ 按类型筛选（视频/图片/音频）
  │   ├─ 按状态筛选（全部/已选用/候选/已废弃）
  │   └─ 搜索（镜头编号）
  │
  ├─ 素材网格（3-4 列自适应）
  │   ┌─────────┬─────────┬─────────┬─────────┐
  │   │ 缩略图   │ 缩略图   │ 缩略图   │ 缩略图   │
  │   │ Shot 03 │ Shot 03 │ Shot 07 │ Shot 12 │
  │   │ 候选 #2  │ 候选 #1  │ 候选 #1  │ 候选 #3  │
  │   │ ✅ 选用  │          │ ✅ 选用  │          │
  │   │ 5.2s    │ 4.8s    │ 6.1s    │ 4.5s    │
  │   │ 12MB    │ 11MB    │ 14MB    │ 10MB    │
  │   └─────────┴─────────┴─────────┴─────────┘
  │
  ├─ 选中后操作栏（底部浮现）
  │   [预览] [选用] [对比] [下载] [删除] [导出 FCPXML]
  │
  └─ 磁盘用量统计
      已用: 1.2GB / 10GB  |  视频 45 个  |  图片 120 个
```

### 2.3 素材卡片设计

对齐现有 `.mini-shot-card` 模式：

```css
.asset-card {
  /* 复用 .mini-shot-card 基础样式 */
  /* 新增字段: 候选编号、seed 值、选用标记 */
}

.asset-card.selected {
  /* 选用态: 紫色边框 + ✅ 角标 */
  border-color: rgba(124, 58, 237, 0.5);
  box-shadow: 0 0 16px rgba(124, 58, 237, 0.15);
}

.asset-card .asset-thumb {
  /* 视频缩略图 — 取第一帧 */
  aspect-ratio: 9/16;  /* 竖屏默认 */
  background: var(--surface-2);
}

.asset-card .candidate-badge {
  /* 候选编号角标 */
  position: absolute;
  top: 0.4rem;
  right: 0.4rem;
  padding: 0.15rem 0.4rem;
  border-radius: 999px;
  background: rgba(0,0,0,0.65);
  color: var(--muted);
  font-size: 0.68rem;
}
```

### 2.4 视频预览播放器

```
点击素材卡片 → 展开预览 Modal:
  ┌─ 视频预览 Modal ─────────────────────────────────┐
  │                                                    │
  │  ┌──────────────────────────────────────────────┐  │
  │  │                                              │  │
  │  │           <video> 播放器                      │  │
  │  │           可循环 / 可倍速 / 可逐帧            │  │
  │  │                                              │  │
  │  └──────────────────────────────────────────────┘  │
  │                                                    │
  │  Shot 03 · 候选 #2 · 5.2s · seed=99                │
  │  生成时间: 2026-04-28 14:23 · 模型: seedance-1.0   │
  │                                                    │
  │  [⏮ 上一候选] [选用此版本] [⏭ 下一候选] [✕ 关闭]    │
  └────────────────────────────────────────────────────┘
```

### 2.5 对比模式

```
选择 2-4 个候选 → [对比] → 并排播放:
  ┌──────────┬──────────┐
  │ 候选 #1   │ 候选 #2   │
  │ seed=42  │ seed=99  │
  │ [视频A]   │ [视频B]   │
  │          │          │
  │ ⭐⭐⭐   │ ⭐⭐⭐⭐  │
  │ [选用]    │ [选用]    │
  └──────────┴──────────┘
  (最多 4 个并排)
```

---

## 三、还需规划的吗？

当前 PRD 已覆盖 **21 项决策**，以下是完整性检查：

| # | 领域 | 状态 |
|---|------|------|
| 1-4 | AI 引擎层 | ✅ 四阶段 + 端点 + 编排 + 范围 |
| 5-7 | 视频导出层 | ✅ 全浏览器 + FCPXML + 零服务端渲染 |
| 8 | 前端 UI | ✅ AgentBoard + DAG + OutputPanel |
| 9 | 角色一致性 | ✅ L1-L3 三层 |
| 10 | 提示词工程 | ✅ 3×3 Contact Sheet + 导演思维 |
| 11-12 | 工程可靠性 | ✅ 3E 迭代 + 三层对账 |
| 13-16 | 商业层 | ✅ 积分计费 + 用户认证 + 内容安全 + 数据分析 |
| 17-18 | 技术约束 | ✅ GET/POST + 管理后台端点 |
| 19 | 电商视频 | ✅ 独立流水线模块 |
| 20-21 | 抽卡 + 素材库 | ✅ 本次补充 |

**可能还需考虑的**:
- 模板系统（预设脚本模板/分镜模板） — 可选，Phase 3+
- WebSocket 实时推送（Agent 状态实时更新，替代轮询） — 可选，Phase 2+
- 多语言 i18n — 暂不需要

---

---

# 第七轮：可视化 DAG + 媒体管理 + AI 可观测性

> 调研日期：2026-04-28
> 参考项目：Langflow (145K⭐)、Sim Studio、MediaCMS、Stash、OpenLIT、Opik、Helicone

## 一、可视化 DAG 工作流编辑器（Phase 2+ 可选）

**参考项目**: Langflow、Flowise、Sim Studio — 均使用 **React Flow** 作为画布引擎。

**dramora 可借鉴**: Phase 2 的 AgentPipeline SVG 硬编码 DAG → 升级为 React Flow 交互式编辑器。

```
Phase 1: SVG 硬编码 DAG（5 节点固定拓扑，只读）
Phase 2+: React Flow 可视化编辑器
  ├─ 拖拽节点调整 DAG 拓扑
  ├─ 点击节点 → 右侧属性面板
  ├─ 实时状态颜色（复用现有状态色）
  ├─ Dagre 自动布局
  └─ 导出/导入 DAG JSON 配置
```

**技术选型**: `@xyflow/react` (React Flow v12, MIT 协议)，与现有 React 技术栈一致。

## 二、媒体资产管理参考

**最佳参考项目**: **Stash** (Go + React + SQLite，10K+⭐)

| 维度 | Stash | dramora |
|------|-------|---------|
| 后端 | Go | Go |
| 前端 | React (嵌入 Go binary) | React (Vite 独立) |
| 数据库 | SQLite | SQLite/PostgreSQL |
| API | GraphQL | REST (GET/POST) |
| 视频管理 | 元数据刮削 + 标签 + 播放 | 素材库 + 候选管理 + 选用/对比 |

**dramora 的 GalleryPage 可以参考 Stash 的**:
- 缩略图网格 + 悬停预览
- 标签/筛选系统
- 批量操作（选用/删除/导出）
- 磁盘用量统计

> MediaCMS (Django+React, 3K⭐) 的 RBAC 权限模型和 FFmpeg 转码流水线也值得参考。

## 三、AI 可观测性（Phase 4 增强）

**当前**: 基础日志 + 轮询。
**目标**: 完整 LLM/视频生成调用链追踪 + 成本归因。

| 工具 | 定位 | dramora 可借鉴 |
|------|------|---------------|
| **OpenLIT** | GPU 监控 + 成本追踪 + OTEL 原生 | Phase 4 集成，监控 LLM 调用链 |
| **Opik** | 评估 + 自动优化成本/延迟 | Agent 质量自动评估 |
| **Helicone** | AI Gateway，语义缓存，省 95% 成本 | 重复 prompt 缓存，降低 API 费用 |
| **Langfuse** | 追踪 + Prompt 管理 + 评估 | LangGraph 原生支持 |

**推荐方案**: Phase 4 集成 **OpenLIT**（最小侵入，OTEL 原生，一行 `openlit.init()`）+ **语义缓存**（相同 prompt 不重复调用 LLM）。

## 四、更新后的决策

| # | 决策 | 选项 |
|---|------|------|
| 22 | **DAG 编辑器** | **Phase 1 SVG 硬编码 → Phase 2+ React Flow 交互式编辑器** |
| 23 | **AI 可观测性** | **Phase 4 集成 OpenLIT OTEL 追踪 + 语义缓存降低 API 成本** |
| 24 | **模板系统** | **镜头风格模板 + 分镜模板 + 提示词模板；创作者可保存/复用/分享** |
| 25 | **批量生成** | **一键生成整集全部镜头；队列可视化 + 预计等待时间；完成后通知** |
| 26 | **新手引导** | **首次使用分步引导 + 示例项目可体验；降低"不知道怎么开始"的摩擦** |

---

---

# 第八轮：AIGC 创作者视角审视

> 站在真实创作者角度审视整个规划，找出"技术上能做但创作者用不起来"的断层。

## 一、创作者最痛的 5 个问题（对照 dramora 现状）

| 创作者痛点 | 行业现状 | dramora 现状 | 差距 |
|-----------|---------|-------------|------|
| **不知道怎么开始** | 橙星梦工厂有拖拽式积木创作、Coze 有模板 | 冷启动空白页，需要创作者自己知道怎么填 | 缺模板+引导 |
| **生成排队到天荒地老** | Seedance 2.0 排队 11 小时 | 无队列可视化，不知道前面排了多少 | 缺队列+通知 |
| **一集要手动点 12 次生成** | Coze 支持全自动批量跑批 | 每个镜头手动点一次生成 | 缺批量 |
| **同一角色每镜长得不一样** | 行业公认 #1 难点，返工 40-50% | L1-L3 已规划但未实现 | 已有方案待实现 |
| **花了多少积分心里没数** | G2 报告 16.7% 用户吐槽积分不透明 | 积分系统已规划但缺消耗预估 | 缺预估+流水 |

## 二、模板系统

**核心思路**：创作者不应该从零开始。系统提供预设模板，创作者可保存自己的模板，形成复利。

### 2.1 三类模板

```
模板类型:
  ├─ 镜头风格模板
  │   ├─ 古风仙侠 · 水墨质感
  │   ├─ 赛博朋克 · 霓虹夜景
  │   ├─ 现代都市 · 日系清新
  │   ├─ 韩漫风 · 高饱和色彩
  │   └─ ...（管理员可扩展）
  │
  ├─ 分镜结构模板
  │   ├─ 3×3 Contact Sheet（9 镜电影级）
  │   ├─ 快手短剧（5 镜快节奏）
  │   ├─ 电商带货（3 镜 Hook→卖点→CTA）
  │   └─ ...（创作者可保存自己的分镜模板）
  │
  └─ 提示词模板
      ├─ 角色描述模板（锚点 + 色板 + 特征）
      ├─ 镜头提示词模板（景别 + 运镜 + 光效）
      └─ 创作者可保存 → 下次一键填入
```

### 2.2 UI 交互

```
StoryAnalysisPage / ShortVideoPage 顶部:
  ┌─ 模板选择栏 ──────────────────────────────────────┐
  │  [古风仙侠] [赛博朋克] [现代都市] [韩漫] [我的模板]  │
  │                                                     │
  │  选择模板后自动填入:                                 │
  │  · 镜头风格预设                                      │
  │  · 分镜数量与结构                                    │
  │  · 默认提示词模板                                    │
  └─────────────────────────────────────────────────────┘
```

## 三、批量生成 + 队列可视化

### 3.1 批量操作

```
StoryboardPage:
  ├─ 选择镜头（checkbox 多选）
  │   ├─ [全选] [选择未生成] [选择失败的]
  │   └─ [批量生成选中镜头]  ← 一键触发
  │
  └─ 每个 ShotCard 右上角显示队列序号:
      Shot 03 ┌────┐
              │ #2 │  ← 队列中第 2 个
              └────┘
```

### 3.2 队列可视化

```
HomePage / StoryboardPage 顶部 — 生成队列面板:
  ┌─ 生成队列 ─────────────────────────────────────────┐
  │  当前队列: 8 个任务  |  预计完成: 约 4 分钟           │
  │                                                     │
  │  ✅ Shot 01  · 候选 #1     已完成 (1.2s)            │
  │  ✅ Shot 01  · 候选 #2     已完成 (1.1s)            │
  │  ◉ Shot 01  · 候选 #3     生成中... 68%            │
  │  ○ Shot 02  · 候选 #1     排队中   #4              │
  │  ○ Shot 02  · 候选 #2     排队中   #5              │
  │  ○ Shot 03  · 候选 #1     排队中   #6              │
  │  ...                                                │
  │                                                     │
  │  完成后 [🔔 通知我]                                   │
  └─────────────────────────────────────────────────────┘
```

### 3.3 通知

```
生成完成 → 多渠道通知:
  1. 页面内 Toast + 音效（默认）
  2. 浏览器 Notification API（需授权）
  3. 页面标题闪烁 "✅ 生成完成 — Dramora Studio"
  
  用户不需要盯着屏幕等。
```

## 四、新手引导

### 4.1 首次使用流程

```
新用户注册 → HomePage:
  ┌─ 欢迎卡片 ────────────────────────────────────────┐
  │                                                    │
  │  🎬 欢迎使用 Dramora Studio！                       │
  │                                                    │
  │  3 步开始你的第一个 AI 漫剧:                         │
  │                                                    │
  │  ✅ 1. 创建一个项目                                 │
  │  ○ 2. 输入故事文本 → AI 自动解析                     │
  │  ○ 3. 一键生成分镜视频                               │
  │                                                    │
  │  [开始引导]  [先自己看看]  [体验示例项目]             │
  └────────────────────────────────────────────────────┘
```

### 4.2 示例项目

```
系统内置一个完整示例项目:
  · 项目: "示例：天门试炼"
  · 已解析的故事分析结果
  · 已生成的 4 个角色 + 4 个场景
  · 已生成的 3 个示例镜头（含视频）

新用户可直接播放示例视频，理解完整流程。
点击"复制为我的项目" → 基于示例开始创作。
```

## 五、其他创作体验优化

### 5.1 积分消耗预估

```
点击"生成视频"前 → 弹窗确认:
  ┌─ 确认生成 ─────────────────────────────┐
  │                                         │
  │  将生成 3 个候选视频                      │
  │  每个候选 5 秒 · 720p                    │
  │                                         │
  │  预计消耗: 300 积分                       │
  │  当前余额: 1,850 积分                     │
  │  生成后剩余: 1,550 积分                   │
  │                                         │
  │  [取消]  [确认生成]                       │
  └─────────────────────────────────────────┘
```

### 5.2 草稿自动保存

```
每 30 秒自动保存:
  · 未提交的剧本编辑内容
  · 调整中的分镜参数
  · 剪辑时间轴草稿

浏览器崩溃/关闭 → 下次打开恢复草稿（localStorage/localforage）
```

### 5.3 风格预设注入

```
创建项目时选择视觉风格 → 自动注入到:
  ├─ 角色生成 prompt（风格关键词自动追加）
  ├─ 场景生成 prompt
  ├─ 分镜提示词模板
  └─ 视频生成参数（模型/分辨率/帧率推荐）
```

## 六、对标竞品的关键差异化

| 能力 | 橙星梦工厂 | 快手漫剧专家 | Coze 模板 | **dramora** |
|------|-----------|------------|----------|------------|
| 多 Agent 协作 | 8 Agent | 无 | 无 | **15 Agent DAG** |
| 人工审批 | 无 | 无 | 无 | **ApprovalGate** |
| 自定义端点 | 锁死供应商 | 锁死供应商 | 锁死供应商 | **管理后台自由配置** |
| 全浏览器端剪辑 | ❌ | ❌ | ❌ | **WebCodecs + FCPXML** |
| 抽卡候选 | 手动 | 手动 | 全自动 | **3 候选生成 + 对比挑选** |
| 模板系统 | ✅ 积木拖拽 | ✅ | ✅ | **三类模板可保存复用** |
| 批量生成 | ❌ | ❌ | ✅ | **一键批量 + 队列可视化** |
| 电商视频 | ❌ | ❌ | ❌ | **独立 ShortVideo 流水线** |
| 开源/自部署 | ❌ | ❌ | 部分 | **完全开源 + SQLite 零配置** |

---

## 七、最终完整性确认

7 轮调研，26 项决策，面向 **AIGC 创作者** 审视后的补充：

```
已有           = 23 项 (AI引擎/产品/商业/工程)
本次新增       =  3 项 (模板系统/批量生成/新手引导)

总决策         = 26 项
PRD 总行数     = ~2100 行
调研覆盖项目   = 50+ 个开源项目
```

**依然可以留到未来的**（不阻塞 MVP）:
- 团队协作/多人项目 — Phase 4+
- 一键发布到平台 — Phase 4+
- A/B 测试视频版本 — 未来
- 素材市场/社区 — 未来
- 移动端 — 未来

---

---

# 第九轮：最终缺口补齐

> 从工程可交付角度审视，补充 API 规范、部署、错误处理等基础设施。

## 一、API 响应格式规范

所有 API 响应遵循统一 envelope 格式：

```json
// 成功
{
  "data": { ... }
}

// 列表
{
  "data": [ ... ]
}

// 错误
{
  "error": {
    "code": "INSUFFICIENT_CREDITS",
    "message": "积分余额不足，需要 300 积分，当前余额 150 积分"
  }
}
```

**HTTP 状态码约定**:

| 状态码 | 场景 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未登录 |
| 402 | 积分不足 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 422 | 参数校验失败 |
| 429 | 请求频率超限 |
| 500 | 服务器内部错误 |
| 502 | 上游 AI 端点错误 |
| 503 | 端点未配置 |
| 504 | 上游 AI 端点超时 |

## 二、Docker 部署

```yaml
# docker-compose.yml (项目根目录)
version: '3.8'
services:
  dramora:
    build: .
    ports:
      - '8080:8080'
    volumes:
      - ./data:/app/data        # SQLite 数据库 + 媒体文件
    environment:
      - DRAMORA_ENV=production
      - DRAMORA_DATA_DIR=/app/data
      # 端点配置在管理后台设置，不走环境变量
      # 唯一需要的环境变量是 DATABASE_URL（可选，有则走 PostgreSQL）
    restart: unless-stopped
```

```dockerfile
# Dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o dramora ./cmd/dramora

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/dramora /usr/local/bin/dramora
COPY --from=builder /app/apps/studio/dist /app/static
EXPOSE 8080
CMD ["dramora"]
```

**一键部署**: `git clone && docker compose up -d` → 访问 `http://localhost:8080`

## 三、每个组件的 4 态设计规范

所有数据驱动组件必须覆盖四种状态：

| 状态 | CSS 类 | 触发条件 |
|------|--------|---------|
| **loading** | 骨架屏 `.skeleton` / spinner | 数据加载中 |
| **empty** | `.empty-board` | 无数据（首次使用） |
| **error** | `.error-board` + 错误信息 + 重试按钮 | 加载失败 |
| **success** | 正常渲染 | 数据就绪 |

```
AgentBoard:
  loading  → 5 个骨架卡片 (animate-pulse)
  empty    → "启动故事解析后，Agent 执行状态将在此展示" + 引导按钮
  error    → "加载 Agent 状态失败" + [重试]
  success  → 5 个 AgentCard（正常渲染）

GalleryPage:
  loading  → 12 个骨架缩略图
  empty    → "还没有生成素材，去分镜工作台开始创作" + [前往]
  error    → "加载素材失败" + [重试]
  success  → 素材网格

ShortVideoPage:
  empty    → 商品输入表单（正常显示，这是初始状态）
  loading  → 脚本生成中（AgentCard running 态）
  error    → "脚本生成失败" + 错误信息 + [重试]
  success  → 脚本预览 + 视频播放器
```

## 四、示例项目数据

```go
// internal/seed/demo.go — 首次启动时自动创建
func SeedDemoData(repo ProjectRepository) {
    // 如果已有项目 → 跳过
    // 无项目 → 创建示例项目 "天门试炼"
    //   ├─ 示例故事源（500 字小说片段）
    //   ├─ 示例角色: 剑仙 / 魔尊 / 师尊 / 小师妹
    //   ├─ 示例场景: 山门 / 试炼场 / 密室
    //   └─ 说明文字标注"这是示例数据，可以删除"
}
```

## 五、快捷键（Phase 2+）

```
Space    播放/暂停视频预览
← →      逐帧前进/后退
Ctrl+Z   撤销编辑
Ctrl+S   保存
Ctrl+Enter 确认/提交
Esc      关闭弹窗/面板
/        聚焦搜索框
```

## 六、最终决策汇总（补至 30 项）

---

# 最终决策汇总 (Master)

> 以下为全规划的唯一权威决策表。文中其他分散表格均为历史记录。

| # | 领域 | 决策 |
|---|------|------|
| 1 | 实施节奏 | 全四阶段规划，分批实施 |
| 2 | AI 端点 | 前端管理后台配置，存 DB 即时生效；4 能力独立端点 + 积分单价；无内置 Provider |
| 3 | 编排模型 | 混合：DAG（生产流水线）+ Blackboard（审查层） |
| 4 | Phase 1 范围 | 基础设施 + 替换现有 5 Agent |
| 5 | Web 剪辑 | 全浏览器端：OpenVideo + WebCodecs + ffmpeg.wasm，零服务端渲染 |
| 6 | 导出格式 | FCPXML 单一桥接 → 剪映/PR/FCPX/达芬奇/CapCut |
| 7 | 人物关系图谱 | Phase 2 增强 StoryMap 为交互式关系图谱 |
| 8 | 前端 UI | AgentBoard + AgentPipeline(DAG) + AgentOutputPanel + BlackboardView；6 个新页面 |
| 9 | 角色一致性 | Character Bible (L1) → IP-Adapter FaceID (L2) → LoRA 微调 (L3) |
| 10 | 分镜提示词 | 3×3 Contact Sheet + 导演思维镜头语言 + Character Anchor System |
| 11 | 迭代优化 | MAViS 3E 原则 — Explore→Examine→Enhance 替代一次性生成 |
| 12 | 任务可靠性 | DB↔Job 三层对账 + Watchdog 巡检（参考 waoowaoo） |
| 13 | 积分计费 | Capability 组合定价 + SHADOW→ENFORCE 灰度 + 兑换码系统 + 管理后台充值 |
| 14 | 用户与权限 | 系统级 admin/user 两级 + Workspace 内 owner/admin/member/viewer 四角色；对象级权限；单用户无感 |
| 15 | 内容安全 | Phase 3 Safety Agent + 第三方审核 API |
| 16 | 数据分析 | Phase 4 创作者数据看板 |
| 17 | API 约束 | 仅 GET/POST；写操作用 POST + :action 后缀；禁止 PUT/PATCH/DELETE |
| 18 | 自定义端点 | 管理后台配置 chat/image/video/audio 四个端点 + 积分单价；某能力未配不可用 |
| 19 | 电商短视频 | 独立流水线；商品→脚本→分镜→视频+配音+字幕；ShortVideoPage 新页面 |
| 20 | 抽卡机制 | 生成 3-5 候选 → 用户挑选最佳 → 重抽扣积分；未选中保留可回溯 |
| 21 | 素材库 | GalleryPage；按项目/镜头分组；预览/对比/选用/批量导出；支持 FCPXML |
| 22 | DAG 编辑器 | Phase 1 SVG 硬编码 → Phase 2+ React Flow 交互式拖拽编辑 |
| 23 | AI 可观测性 | Phase 4 集成 OpenLIT OTEL 追踪 + 语义缓存降本 |
| 24 | 模板系统 | 三类模板：镜头风格/分镜结构/提示词模板；创作者可保存复用 |
| 25 | 批量生成 | 一键整集生成 + 队列可视化 + 预计时间 + 浏览器通知 |
| 26 | 新手引导 | 分步引导 + 内置示例项目（可复制）；降低上手摩擦 |
| 27 | API 规范 | 统一 `{data}` / `{error: {code, message}}` envelope；11 种 HTTP 状态码 |
| 28 | Docker 部署 | `docker compose up -d` 一键启动；SQLite 零配置 |
| 29 | 四态设计 | 每个组件覆盖 loading / empty / error / success |
| 30 | 示例数据 | 首次启动自动 seed 示例项目 |
| 31 | CI/CD | GitHub Actions: go test + build + lint → Docker |
| 32 | 测试分层 | Domain单元 → Handler集成 → Repo集成 → Adapter mock → E2E |
| 33 | 安全 | API Key AES-256-GCM 加密；频率限制；输入校验；日志脱敏 |
| 34 | 数据库 | SQLite 默认（WAL 模式）；DATABASE_URL 配了自动切 PostgreSQL |
| 35 | 兑换码 | 格式 DRAM-XXXX-XXXX-XXXX；批量生成+CSV导出；乐观锁防超兑；核销追踪 |
| 36 | 通知 | 浏览器 Notification API + Toast；生成完成后通知 |

---

# 第十轮：CI/CD + 测试 + 安全

## 一、CI/CD Pipeline

```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]
jobs:
  backend:
    steps:
      - run: GOTOOLCHAIN=local go test ./...
      - run: GOTOOLCHAIN=local go build ./...
      - run: test -z "$(gofmt -l .)"
  frontend:
    steps:
      - run: cd apps/studio && npm ci && npm run build && npm run lint
  docker:
    if: github.ref == 'refs/heads/main'
    needs: [backend, frontend]
    steps:
      - run: docker build -t dramora .
```

## 二、测试分层

| 层级 | 工具 | Phase 1 要求 |
|------|------|-------------|
| Domain 单元 | `go test` | 状态机转换 + ID 生成 |
| Handler 集成 | `httptest` | HTTP 状态码 + JSON envelope |
| Repository 集成 | SQLite 真实 DB | CRUD + 约束校验 |
| Adapter 单元 | mock HTTP server | 重试 + 超时 + 错误映射 |
| Agent 单元 | mock adapter | Prompt 渲染 + JSON 解析 |

## 三、安全

| 措施 | 实现 |
|------|------|
| API Key | DB 中 AES-256-GCM 加密，日志脱敏 |
| 频率限制 | 60次/分钟（生成类 10次/分钟） |
| 输入校验 | 故事源 ≤20,000字；图片 ≤10MB |
| XSS/CSRF | HTML 转义 + SameSite Cookie（Phase 2） |
| SQL 注入 | 参数化查询（driver 层面保证） |

---

**最终: 33 项决策, ~2500 行 PRD, 10 轮迭代, 50+ 开源项目调研。规划完整，可进入实现。**
