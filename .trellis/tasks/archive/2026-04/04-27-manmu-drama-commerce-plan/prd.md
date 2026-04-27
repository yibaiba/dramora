# brainstorm: 漫幕 AI 生成漫剧平台规划

## Goal

规划「漫幕」作为 AI 生成漫剧平台的第一版产品与技术路线：用 LLM 生成故事、人物设定、分镜和提示词，对接图片/视频/配音/字幕生成模型，形成从创意到漫剧成片的工作流。后端优先使用 Go，前端使用 React。

## What I already know

- 用户明确修正方向：不是传统漫剧 + 电商网站，而是生成漫剧的平台。
- 平台需要对接视频生成模型和 LLM 模型。
- 需要参考优秀漫剧/AI 视频/AI 创作平台。
- Go 后端和 React 前端仍是偏好技术栈。
- 当前仓库主要是 Trellis 脚手架，尚无成型业务代码，可以从零设计架构。

## Research Notes

### Product references

- Runway: 适合参考专业视频生成、镜头级生成、时间线编辑、多镜头创作体验。
- Pika: 适合参考快速生成、轻量动画、创作者友好的 prompt-to-video 体验。
- Kling / 可灵: 适合参考中文市场、高质量文生视频/图生视频、真实感镜头。
- Luma Dream Machine: 适合参考真实感、动作表现、图生视频体验。
- PixVerse: 适合参考短视频、社媒、快迭代生成体验。
- Krea / StoryboardHero 类产品: 适合参考脚本到故事板、角色一致性、多格漫画/分镜工作流。
- 国内漫剧/漫画/短剧平台需要继续人工核验真实产品细节；当前可作为方向参考的是：AI 文生漫、分镜生成、角色库、配音合成、模板化发布。

### Open-source / engineering references

详见 `research/ai-manju-generation.md`。

- ComfyUI: 模块化节点/图工作流，是 AI 生成管线编排的重点参考。
- CogVideo / CogVideoX: 文生视频、图生视频模型参考。
- Wan2.1: 中文生态中重要的视频生成模型参考。
- HunyuanVideo: 腾讯混元视频生成模型参考。
- Dify / Flowise: 视觉化 AI workflow、agent、模型接入、RAG、Prompt IDE 的产品参考。
- Ollama / LangChainGo: Go 生态下接入本地/远程 LLM 的工程参考。
- LiteLLM / Langfuse: 统一模型网关、成本追踪、prompt/version/trace/observability 参考。

## Product Positioning

「漫幕」应定位为 **AI 漫剧 Studio**，不是普通内容站：

- 输入：故事想法、小说片段、角色设定、图片素材、参考风格。
- 中间产物：故事分析、世界观、人物关系、人物卡、人物参考图、场景地图、场景概念图、道具卡、剧情大纲、分集脚本、分镜表、镜头 prompt、角色/场景一致性素材、配音文案、字幕。
- 输出：动态漫/漫剧视频、分镜图、封面海报、短视频切片、发布素材。
- 管理：项目、资产、模型任务、版本、审核、导出、发布。
- 生产方式：多 Agent 协作，不让单个大模型包办全部创作。

## Assumptions (temporary)

- MVP 先做 Web Studio，而不是 C 端播放社区。
- MVP 先做「AI 生成 + 完整在线剪辑」闭环，允许人工调整分镜、prompt、镜头、配音、字幕、转场和导出。
- MVP 先接入外部模型 API，最快验证产品闭环；暂不维护自托管 GPU，但保留 provider adapter 抽象，后续可接 ComfyUI/GPU worker。
- MVP 视频生成任务可能耗时较长，必须以异步任务、队列、进度状态、失败重试和成本记录为核心。
- MVP 重点是「脚本 -> 分镜 -> 画面/视频 -> 时间线剪辑 -> 配音/字幕 -> 转场 -> 站内导出」闭环。
- 电商/内容社区可以作为后续变现，不进入第一阶段核心。

## Requirements (evolving)

- 使用 Go 构建平台 API、任务编排、模型网关适配、资产管理和权限系统。
- 使用 React + TypeScript 构建 AI Studio 前端。
- 支持项目工作区：每个漫剧项目包含故事、角色、分集、分镜、资产、生成任务和导出结果。
- 支持 LLM 编剧：
  - 一句话创意生成故事大纲。
  - 对长文本/小说片段做故事分析：主题、冲突、人物关系、情绪曲线、场景、关键事件。
  - 生成角色设定、世界观、分集剧情、分镜脚本。
  - 将分镜脚本转换为图片/视频模型 prompt。
- 支持 C/S/P 资产编号：
  - Cxx = Character / 角色资产。
  - Sxx = Scene / 场景资产。
  - Pxx = Prop / 道具资产。
  - 编号用于提示词引用、镜头引用、导出包和连续性检查。
- 支持 AI 生成人物：
  - 从故事中自动抽取主要/次要角色。
  - 生成人物卡：姓名、身份、年龄段、性格、动机、外貌、服装、口头禅、关系网。
  - 生成人物参考图：全身图、三视图、表情包、常用姿势。
  - 人物参考图作为后续分镜图和视频镜头的输入资产，减少角色漂移。
  - 允许人工锁定人物设定，避免后续 LLM 自动覆盖已确认角色。
- 支持 Scene Map / 场景资产：
  - 从故事中自动抽取场次、地点、时间、气氛、天气、光线、色调、镜头情绪和关键道具。
  - 生成场景卡：名称、地点、所属区域、时间段、氛围、光线、色彩、连续性备注。
  - 生成场景概念图：大远景/建立镜头、关键背景板、常用角度、必要的光照/气氛变体。
  - Shot 必须引用 `scene_id`，而不是每个镜头重复粘贴场景描述。
  - 镜头 prompt 生成时自动注入场景概念图、锁定角色参考图和道具参考图，降低“不连戏”。
- 支持道具资产：
  - 从故事和分镜中提取高频/关键道具。
  - 生成道具卡与参考图，供镜头生成复用。
- 支持候选生成与锁定：
  - 人物、场景、道具、关键帧都应支持一次生成多个候选。
  - 用户选择一个版本锁定后再进入昂贵的视频生成。
  - 借鉴 Openjourney 的候选网格和 AIComicBuilder 的分镜看板。
- 支持模型适配层：
  - LLM provider：OpenAI-compatible、本地 Ollama、国产模型 API。
  - Image provider：Stable Diffusion / ComfyUI / 第三方图片生成 API。
  - Video provider：Runway/Pika/Kling/Luma/PixVerse API 或自托管 CogVideo/Wan/HunyuanVideo。
  - Audio provider：TTS、角色配音、背景音乐可后续扩展。
  - Provider capability metadata：T2V、I2V、首尾帧视频、视频续写、最大参考图数量、最大参考视频数量、最大时长、提示词语言偏好、商业/许可证限制。
- 支持多 Agent 协作生产线：
  - Producer Agent：控制目标、预算、时长、阶段计划和审批点。
  - Story Analyst Agent：分析故事主题、冲突、情绪曲线、人物、场景、道具。
  - Screenwriter Agent：生成分集大纲、场景拆解、台词和旁白。
  - Character Designer Agent：生成人物卡、服装变体、全身/三视图/表情/姿势提示词。
  - Scene Designer Agent：生成场景卡、地点地图、光线/氛围、场景概念图提示词。
  - Prop Designer Agent：提取并设计关键道具。
  - Storyboard Agent：生成镜头卡、时长、动作、台词、角色/场景/道具引用。
  - Prompt Engineer Agent：把结构化资产转换为 provider-specific prompts。
  - Director / Cinematographer Agents：决定镜头路线、构图、运镜、首尾帧策略。
  - Voice & Subtitle Agent：生成配音脚本、角色声音方向、字幕时间段。
  - Editor Agent：组装时间线、裁剪建议、转场选择。
  - Continuity Supervisor Agent：检查人物、服装、场景、道具和叙事连续性。
  - Safety & Copyright Agent：检查提示词和产物风险。
  - Cost Controller Agent：预估/记录模型成本，防止超预算生成。
- 支持异步生成任务：
  - 覆盖 workflow、node、agent run、generation job 四层状态机。
  - 支持排队、预检、提交、轮询、下载、后处理、成功、失败、取消、重试、预算暂停。
  - 记录 prompt、模型、参数、成本、耗时、输入输出资产、job attempts、状态事件。
- 支持分镜编辑：
  - 镜头列表 / 时间线 / 卡片式 storyboard。
  - 每个镜头可编辑画面描述、对白、动作、镜头语言、模型参数。
  - 允许重生成单个镜头，而不是整片重跑。
- 支持资产库：
  - 故事分析、人物卡、人物参考图、场景地图、场景概念图、道具卡、道具参考图、分镜图、视频片段、音频、字幕。
  - 资产版本与来源追踪。
- 支持导出：
  - 分镜 PDF/图片包。
  - 视频片段下载。
- 支持完整在线剪辑器：
  - 多轨时间线：视频、图片/keyframe、音频/TTS、字幕/文本。
  - 基础剪辑：拖拽、裁剪、分割、删除、复制、排序。
  - 基础转场：fade、crossfade/dissolve、slide。
  - 预览播放器与时间线同步。
  - 字幕编辑：文本、时间、样式预设。
  - 配音面板：按台词/场景生成 TTS 音频。
  - 站内 MP4 导出任务。
- 支持 React Studio 工作台：
  - 项目列表、项目总览、Episode command center、脚本/故事分析、资产库、人物/场景/道具 workshop、Storyboard Kanban、Agent Board、任务队列、时间线编辑器、导出页。
  - 桌面端采用左侧导航 + 中央工作区 + 右侧 inspector + 底部任务/成本 rail。
  - 移动端优先支持审批、候选查看、任务监控和导出下载；复杂时间线编辑以桌面为主。

## Acceptance Criteria (evolving)

- [x] 明确 MVP 是「生成工作流平台」，而非传统电商/播放站。
- [x] 明确第一阶段支持哪些模型类型：LLM、图片、视频、TTS、合成。
- [x] 明确第一阶段模型接入方式：外部 API 优先，成本按调用计费，暂不维护 GPU。
- [x] 明确 MVP 输出级别：完整在线剪辑器，包含时间线剪辑、配音、字幕、转场、站内导出。
- [x] 明确故事分析和人物生成链路：故事解析、角色卡、增强人物参考图、角色锁定、分镜引用。
- [x] 明确 Scene Map 链路：场次/地点/气氛抽取、场景卡、场景概念图、镜头引用、连续性约束。
- [x] 明确多 Agent 协作链路：故事分析、编剧、人物、场景、导演、提示词、剪辑、连续性质检、成本控制。
- [x] 明确 GitHub AI 漫剧/视频项目映射：AIComicBuilder、AIYOU、Seedance2、Openjourney、Wan2.1、CogVideoX、AnimateDiff、MuseTalk/Wav2Lip。
- [x] 明确 Go 后端核心领域模型：项目/剧集、故事、人物/场景/道具版本、镜头、资产 lineage、workflow/agent、模型任务、时间线导出。
- [x] 明确 Studio 核心页面：项目列表、项目工作台、脚本编辑、分镜编辑、时间线剪辑器、任务队列、资产库、导出页。
- [x] 明确 Go 后端模块边界、任务状态机、核心数据库实体。
- [x] 明确 React 前端技术形态与画布/时间线/分镜组件策略。

## Definition of Done (team quality bar)

- PRD 已更新为 AI 生成漫剧平台方向。
- 平台参考和开源工程参考已归档到 `research/`。
- MVP 范围和 out-of-scope 明确。
- 下一阶段可以进入架构设计、领域模型和接口边界规划。

## Out of Scope (explicit)

- 第一阶段不做传统电商闭环。
- 第一阶段不做 C 端内容社区和复杂推荐。
- 第一阶段不承诺一键生成完整高质量长片。
- 第一阶段不默认自研视频基础模型。
- 第一阶段不默认多人实时协作。
- 第一阶段不做 Premiere/CapCut 级高级剪辑能力；只做能完成漫剧成片的窄版完整编辑闭环。

## Technical Approach

推荐以 **Go 控制平面 + 外部模型 API 执行平面 + React Studio** 开始：

- Backend API: Go。
- Backend API implementation defaults:
  - Router: Chi。
  - DB access: pgx + sqlc。
  - Migrations: golang-migrate。
  - Queue: River + PostgreSQL transactional jobs。
  - Realtime: SSE first, WebSocket later if needed。
  - API contract: OpenAPI YAML first, codegen after route contracts stabilize。
- Backend shape:
  - `internal/domain`：纯领域模型和枚举。
  - `internal/service`：用例/应用服务。
  - `internal/repo`：PostgreSQL repository。
  - `internal/provider`：LLM/image/video/audio provider adapters。
  - `internal/workflow`：workflow graph、agent orchestration、approval gates。
  - `internal/media`：对象存储、缩略图、FFmpeg/export helpers。
- Worker / scheduler: Go worker 管理任务状态、队列、重试、成本、回调；MVP 倾向 PostgreSQL source of truth + River transactional enqueue，Redis/Asynq 可作为备选执行层。
- Model execution:
  - MVP：通过 provider adapter 调用外部 LLM、图片、视频、TTS API。
  - Later：同一 provider adapter 层可扩展到 ComfyUI API、Python inference service 或独立 GPU worker。
- Database: PostgreSQL。
- Queue/cache: MVP 倾向 River + PostgreSQL transactional jobs；Redis 可用于缓存、限流、realtime fanout，Asynq 可作为 Redis 队列备选。
- Storage: S3-compatible object storage；国内部署可替换为 OSS/COS。
- Realtime status: WebSocket 或 SSE 推送生成进度。
- Observability: 记录每次模型调用的 prompt、参数、token、耗时、成本、错误。
- Frontend: React + TypeScript。
- Public/landing: Next.js React 更适合 SEO 和品牌展示。
- Studio app: Vite React 或 Next.js app route 均可，重点是复杂交互性能。
- UI: Tailwind + shadcn/ui/Radix + Lucide icons。
- Storyboard/timeline:
  - Storyboard 用于镜头生成前的结构化规划。
  - Timeline editor 是 MVP 核心，可参考 VNVE、FreeCut、Twick、DesignCombo React Video Editor。
  - React Flow 可用于后续剧情分支/工作流图。
  - Excalidraw/tldraw 可用于后续自由画布；tldraw 商业许可需评估。
- Render/export:
  - MVP 决定采用混合路线：前端实时预览，服务端导出 MP4，后续补 WebCodecs/Remotion。
  - 导出链路：timeline JSON -> export job -> FFmpeg/renderer worker -> MP4。
  - 可参考 Twick 的 browser/server render 分层。
  - 可参考 VNVE 的 PixiJS + WebCodecs 视觉小说视频思路。
  - Remotion 适合 React 组件化渲染，但商业/公司使用需要先审查许可证。
- Story/character generation:
  - LLM 先做故事分析和 story bible，而不是直接生成镜头。
  - Character workshop 是 MVP 核心模块：人物卡、全身图、三视图、表情包、常用姿势、锁定版本。
  - StoryDiffusion 和 AI Comic Generator 的关键启发是：角色一致性与 JSON-driven workflow 应作为底层数据模型。
- Scene/prop generation:
  - Scene Map 是 MVP 核心模块：地点、区域、时间、气氛、光线、色调、场景概念图和连续性备注。
  - Prop Map 用于承载关键道具和可复用物件，镜头生成时与角色、场景一起注入上下文。
  - BigBanana、CineGen、ai-shotlive 的共同启发是：角色定妆照 + 场景概念图 + 道具参考图共同约束镜头，才能减少“不连戏”。
- Shot / storyboard generation:
  - AIComicBuilder 的关键启发是：剧本导入 -> 角色四视图 -> 分镜看板 -> 首尾帧 -> 视频提示词 -> 视频生成 -> FFmpeg 合成。
  - Seedance2 Storyboard Generator 的关键启发是：C/S/P 编号、0-3s / 3-6s / 6-9s 时间轴提示词、尾帧描述和视频续写都应进入数据模型。
  - Wan2.1 / CogVideoX 的关键启发是：provider adapter 需要显式区分 T2V、I2V、首尾帧视频、视频续写和提示词扩写能力。
- Multi-agent orchestration:
  - MVP 内部采用固定生产 SOP，而不是一开始暴露自由拖拽 Agent 工作流。
  - 参考 MetaGPT 的角色/SOP、CrewAI 的 role-specific agents、LangGraph 的 stateful workflow + human-in-the-loop、Flock 的 Human Node/Agent Node、Network-AI 的 shared state/guardrails/budgets。
  - 所有 Agent 读写结构化 artifact，不依赖散乱聊天上下文。
- Studio UX references:
  - Openjourney 的候选网格适合人物、场景、道具、关键帧的多版本选择。
  - AIYOU 的节点画布适合未来高级工作流；源码更像自研 canvas 而非 React Flow，因此漫幕应先定义 renderer-independent workflow graph，MVP 先用固定 SOP + Agent Board。
  - AIComicBuilder 的 Storyboard Kanban 适合镜头生成进度管理。

## Initial Workflow

```text
Idea / Novel
  -> Producer Agent: production plan, budget, approval gates
  -> LLM: story analysis, world bible, character/scene/prop extraction
  -> Image model: character full-body refs, turnarounds, expressions, poses
  -> Image model: scene concept refs, background plates, prop refs
  -> Human lock: selected character reference/version
  -> Human lock: selected scene/prop references
  -> LLM: episode outline
  -> LLM: scene breakdown + shot list
  -> LLM: prompt pack using locked characters and style bible
  -> Director/Cinematographer Agents: shot route, framing, motion
  -> Image model: character refs, keyframes, storyboard frames
  -> Video model: shot clips
  -> Continuity/Safety Agents: review outputs
  -> TTS / subtitle: voice and captions
  -> Timeline editor: trim, split, reorder, transition, captions
  -> Editor Agent: rough cut suggestions
  -> Export worker: render MP4
  -> Human review/edit/regenerate/re-export
```

Studio review surfaces:

- Candidate grid：展示图片/视频候选，支持选择、锁定、重生成。
- Storyboard Kanban：按镜头状态分列，例如 planned、prompt ready、keyframe ready、video generating、review needed、approved。
- Agent Board：展示当前 Agent、等待审批点、成本、错误、连续性质检问题。

## Decision (ADR-lite)

**Context**: 生成漫剧平台的核心难点不是传统 CRUD，而是长耗时、多模型、多资产、多版本的生成工作流。

**Decision**: MVP 采用外部 API 优先策略；平台后端不直接绑定某一个模型，以 provider adapter + async job + asset lineage 作为核心架构。Go 负责业务控制平面，模型推理先通过外部 API 执行，暂不维护自托管 GPU。

**Consequences**:

- 优点：最快验证产品闭环，降低早期 GPU 运维成本，并能快速比较不同视频生成供应商。
- 代价：成本受外部 API 定价影响，供应商可用性/限流/审核策略需要被纳入错误处理。
- 后续：可加入模型评分、A/B prompt、角色一致性评估、高级剪辑效果、多用户协作。

## Technical Notes

- `ui-ux-pro-max` 建议 AI Studio 采用 video-first hero、bold minimalism、高对比和明确进度反馈。
- 生成类 UX 必须突出：步骤进度、队列状态、失败原因、重试入口、成本预估。
- 当前 GitHub 调研中，ComfyUI、CogVideo、Wan2.1、HunyuanVideo、Dify、Flowise、Ollama、LangChainGo、LiteLLM、Langfuse 都值得进一步拆源码/接口。
- 在线剪辑器调研见 `research/web-video-editor.md`；重点参考 VNVE、FreeCut、Twick、DesignCombo React Video Editor、Remotion、ffmpeg.wasm、StoryDiffusion、AI Comic Generator。
- 人物生成/角色一致性应参考 StoryDiffusion 的 consistent image/storyboard 思路，以及 AI Comic Generator 的 Character Workshop、Global Config JSON、Storyboard Editor。
- AI 漫剧项目 feature map 见 `research/ai-manju-project-map.md`；重点参考 BigBanana、CineGen、ai-shotlive 的 Scene/Set Design、Project Resources、Script-to-Asset-to-Keyframe。
- 多 Agent 生产线见 `research/multi-agent-production-map.md`；重点参考 MetaGPT、CrewAI、LangGraph、AutoGen、Flock、Network-AI。
- GitHub AI 漫剧/视频项目追加调研见 `research/github-ai-video-drama-projects.md`；重点参考 AIComicBuilder、AIYOU、Seedance2 Storyboard Generator、Openjourney、Wan2.1、CogVideoX、AnimateDiff、MuseTalk、Wav2Lip。
- AIYOU 节点画布深挖见 `research/aiyou-workflow-canvas-deep-dive.md`；重点参考节点类型、依赖规则、拓扑执行、分镜网格/拆解、视频 provider submit/poll。
- Go 后端领域模型和 PostgreSQL 表设计见 `research/go-backend-domain-model.md`；重点包含 Project/Episode、Character/Scene/Prop versions、Shot、Asset lineage、Workflow/Agent、GenerationJob、Timeline/Export、Review/Cost。
- Agent 工作流状态机、Job 调度和成本控制见 `research/workflow-job-state-machine.md`；重点包含 workflow/node/agent/generation job 状态机、River/Asynq/Temporal/Hatchet/Inngest/LangGraph 调度取舍、预算预留、cost ledger、provider submit/poll/cancel、realtime 事件。
- React Studio 页面结构、Agent Board、Storyboard Kanban、资产库、任务队列和时间线编辑器规划见 `research/react-studio-ui-map.md`。
- Go API 路由与模块脚手架计划见 `research/go-api-scaffold-plan.md`；重点包含 Chi/pgx/sqlc/River 默认栈、`apps/api`/`apps/worker`/`internal/*` 包结构、REST route groups、middleware、DTO/OpenAPI、worker job kinds 和第一实施切片。
- Timeline/editor 技术选型见 `research/timeline-editor-tech-selection.md`；MVP 决定自建窄版内部 timeline editor + 服务端 FFmpeg 导出，借鉴 FreeCut/VNVE/Twick，但暂不直接依赖完整第三方 editor SDK。
- AI 漫剧 GitHub 深挖实现地图见 `research/ai-manju-github-deep-dive-implementation-map.md`；最终组合策略为 AIComicBuilder 式阶段流水线 + AIYOU 式内部 typed workflow graph + Seedance2 式 C/S/P 与时间轴 prompt discipline + Openjourney 式候选 UX。

## Resolved Decisions

- 模型接入策略：外部 API 优先，最快验证产品，成本按调用计费，暂不维护 GPU。
- MVP 输出级别：完整在线剪辑器，支持时间线剪辑、配音、字幕、转场和站内导出。
- 渲染导出路线：混合路线，前端实时预览，服务端导出 MP4，后续补 WebCodecs/Remotion。
- 人物参考图粒度：增强版，全身图 + 三视图 + 表情包 + 常用姿势。
- Scene Map：MVP 必须包含场景卡、地点/区域、时间/天气/气氛/光线、场景概念图、镜头引用和连续性约束。
- 多 Agent：MVP 使用固定生产 SOP + Agent Board + human approval gates；暂不开放自由编排 Agent 工作流。
- 资产流：MVP 必须支持 C/S/P 编号、候选网格、版本锁定、首尾帧、Storyboard Kanban、provider capability metadata。
- 后端领域模型：以 PostgreSQL 为 source of truth；媒体只存对象存储 URI，不在 DB 存 base64；每次生成必须能通过 asset lineage + prompt + provider + params 复现。
- 任务调度：MVP 以 PostgreSQL 中的 workflow/job 状态为业务真相，队列只做执行传输；优先评估 River transactional enqueue，Asynq 保留为 Redis 队列备选。
- Studio UI：MVP 使用固定 SOP 的可视化工作台，不先开放自由 Agent workflow builder；优先做 Agent Board + Storyboard Kanban + 资产候选网格 + 窄版完整时间线编辑器。
- Go API 脚手架：采用模块化单体 + REST API + SSE；业务状态由 PostgreSQL 表驱动，handler/service/repo/workflow/provider/cost/media/realtime 分层。
- Timeline/editor：MVP 自建窄版内部 timeline，后端 `timeline_tracks`/`timeline_clips` 为权威数据；导出走 Go/River/FFmpeg worker，WebCodecs/Remotion/ffmpeg.wasm 作为后续可选增强。
- GitHub 深挖结论：MVP 以 AIComicBuilder 的端到端漫剧流水线为最直接产品骨架，AIYOU 作为未来节点画布参考，Seedance2 作为 prompt/资产编号规则，Openjourney 作为候选生成 UX；Wan/CogVideo/AnimateDiff/MuseTalk/Wav2Lip 作为 provider capability 与后续模型扩展参考。

## Open Questions

- 下一步需要细化领域模型、任务状态机、Studio 页面结构和仓库脚手架。
