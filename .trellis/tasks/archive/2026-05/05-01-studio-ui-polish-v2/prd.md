# Studio UI 打磨 v2

## Goal

Studio 经过多轮功能扩张（PR1-PR9）后，信息密度高、视觉/交互细节散落多处。本任务在不大改架构与 API 的前提下，对前端进行一轮统一的视觉、交互、节奏与响应式打磨，让导演台 / 故事解析 / Storyboard / Assets / Timeline / WorkerMetrics / Auth 等页面体感一致，并补齐键盘可达性与暗色专业感。

## What I already know

- 前端栈：`apps/studio` Vite + React 19 + TS，主入口 `src/App.tsx` + `src/studio/pages/*`。
- 设计基调：暗色专业感、电影感、AI 漫剧生产工作台（不是普通后台）。
- 全局样式集中在 `apps/studio/src/index.css`（已有 ~1700 行，含 `.blackboard-*` / `.board-*` / chip 体系）。
- 已沉淀的共享组件：`ReviewSummaryChips`、`AgentFeedbackWorkspace`、`AnalysisResultCard`、`ProductionFlowPanel`、`RecoveryPanel` 等；chip / 状态色（`current/warn/ready`）已统一。
- 服务端状态全部走 `src/api/hooks.ts`；本任务原则上不新增 API。
- 验证基线：`cd apps/studio && npm run lint && npm run build`，0 errors / build 通过；现有 16 条 react-hooks/exhaustive-deps warnings 是历史基线。

## Assumptions (temporary)

- 不引入新的 UI 组件库（不上 shadcn / radix / antd），保持纯 CSS + 现有 token 的演进路径。
- 不做大规模文件结构重排；改动集中在 `index.css` + 各 page/component 的局部 className/JSX 微调。
- 桌面优先，移动端做到「能用、不破版」即可，不追求精细原生体验。
- 不强行替换图标库；继续用 `lucide-react`。
- 不动后端、不改 OpenAPI、不动 hooks 数据契约。

## Open Questions

- 已确认：design tokens 继续在 `index.css` 顶部 `:root` 集中维护（不抽 `tokens.css`）。
- 已确认：响应式覆盖三档断点 `1280 / 1024 / 768`，<768 也要可用不破版。
- 已确认：键盘可达性范围为「导航 + review queue + 对话框」，暂不覆盖每个表单字段。
- 已确认：本轮同时交付亮色主题（暗色仍是默认），需要 token 同源、用户可切换并持久化。

## Requirements (evolving)

- 抽出 / 整理 design tokens（颜色 / 间距 / 圆角 / 阴影 / 动画时长），在 `:root` 集中声明，去掉散落的硬编码色值。
- 统一卡片体系：`board-card / blackboard-card / review-relay-card` 等多种卡片样式收敛到一致的边框、圆角、内边距、hover/focus 视觉反馈。
- 统一状态色语义：`pending / warning / ready / danger / info` 五档，chip / badge / banner / 按钮派生色都从同一组 token 取。
- 统一交互节奏：hover / active / focus-visible 过渡时长一致；键盘 focus ring 全局可见。
- 顶部导航 / 侧栏整理：在内容密集页面（StoryAnalysis / Storyboard）保证视觉锚点清晰，subnav / chip-row 不与正文挤压。
- 关键页面响应式兜底：≥1280 大屏不浪费横向空间；768~1280 主体可滚动不破版；<768 至少导航可用、卡片纵向堆叠不溢出。
- 键盘可达性：所有主要操作按钮 `:focus-visible` 有可见 ring；j/k 键盘导航推广到 review queue / agent board / worker metrics 表格（按页面适配）。
- 空态与加载态：每个页面的 loading / empty / error 三态有统一的视觉与文案模板（不再各自手写）。
- 不引入回归：现有 lint / build 必须仍 0 errors，warnings 不增。

## Acceptance Criteria (evolving)

- [x] `index.css` 顶部 `:root` 有完整的 design tokens（color / spacing / radius / shadow / motion），现有硬编码色值大头已收敛；亮色主题通过 `:root[data-theme="light"]` 覆盖同名 token，组件代码不感知主题分支。
- [x] 主题切换：侧栏/顶栏新增切换入口（暗 / 亮），偏好持久化到 `localStorage`，初次访问跟随 `prefers-color-scheme` 兜底。
- [x] 卡片体系收敛到 ≤3 种基础卡（普通卡 / 强调卡 / 子卡），其余通过修饰类派生。
- [x] 五档状态色（pending / warning / ready / danger / info）有统一 token，chip / badge / banner / 按钮变体从同一来源取。
- [x] 全局 `:focus-visible` ring 一致可见；至少 HomePage / StoryAnalysisPage / StoryboardPage / AssetsGraphPage / TimelineExportPage / WorkerMetricsPage 主操作按钮可键盘聚焦。
- [x] j/k 键盘导航至少推广到 1 处新场景（如 agent board 或 review queue 主列表），且不与既有 history 列表冲突。
- [x] 6 个主要页面（Home / StoryAnalysis / Storyboard / AssetsGraph / TimelineExport / WorkerMetrics）在 1280 / 1024 / 768 三档断点下不破版，主要内容可读可操作。
- [x] loading / empty / error 三态 UI 在主页面统一（不要求一次性全部页面，但至少 Home + StoryAnalysis + Storyboard + AssetsGraph + TimelineExport 一致）。
- [x] `cd apps/studio && npm run lint && npm run build` 仍 0 errors，warnings 不增（仍然 ≤16）。
- [ ] 关键视觉变更附 before/after 截图或简短说明，便于回看。

## Definition of Done (team quality bar)

- 视觉/交互改动不破坏既有功能（手测 6 个主要页面 happy path）。
- Lint / build 0 errors，warnings 不增。
- 任何新增的 className / token 在 `index.css` 顶部或对应 component 文件就近声明，不留无主样式。
- 改动较大的页面在 commit 信息中标注「视觉打磨」前缀，便于后续 review 定位。
- 不动后端、不动 OpenAPI、不动 `src/api/*` 数据契约。

## Out of Scope (explicit)

- 不引入新的 UI 组件库 / 设计系统（shadcn / radix / antd / mantine 等）。
- 不引入 CSS-in-JS / tailwind / 新构建插件。
- 不重写路由 / 不拆分 `apps/studio` 文件结构。
- 不新增后端 API、不动 hooks 契约。
- 移除"不做亮色主题"的 out-of-scope 项（已纳入本轮）。
- 不做 i18n（沿用现有中文文案）。
- 不做无障碍全量 WCAG 评估，仅覆盖 focus ring + 键盘导航的最小子集。
- 亮色主题颜色仅做"专业可读"基线，不追求与暗色完全镜像的视觉强度。

## Technical Notes

- token 建议命名：`--color-bg-base / --color-bg-elevated / --color-border / --color-text / --color-state-{pending,warn,ready,danger,info} / --space-{1..8} / --radius-{sm,md,lg} / --shadow-{1,2} / --motion-{fast,base}`。
- 卡片体系建议：基础类 `.surface-card`，修饰 `.surface-card--emphasis / --inset / --compact`；现有 `.board-card / .blackboard-card` 内部聚合为修饰，不大改 className 来源。
- focus ring 建议：`outline: 2px solid var(--color-state-info); outline-offset: 2px;` 限定在 `:focus-visible`，避免鼠标点击残留。
- 响应式：使用 `@media (max-width: 1279px) / 1023px / 767px` 三档；侧栏在 <1024 折叠为顶栏抽屉，<768 进入纵向堆叠。
- 键盘导航：复用 `AgentFeedbackWorkspace` 已验证的 `data-id + querySelector + scrollIntoView` 范式，不再用 ref-in-render。
- 空/加载/错误态：抽 `StatePlaceholder` 共享组件（图标 + 标题 + 文案 + 可选 action），所有页面消费同一来源。

## Research Notes

（待补：现状截图清单 / 当前硬编码色值统计 / 各页面已有/缺失的空态实现位置。可在执行阶段逐步补充。）
