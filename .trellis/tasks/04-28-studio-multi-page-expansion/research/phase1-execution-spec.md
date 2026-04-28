# Phase 1 执行规格

> 从 PRD 到代码的桥梁。每个条目可直接转为实现任务。

---

## 一、执行顺序（依赖链）

```
PR1.5 (SQLite) → PR1 (端点适配器) → PR2 (DAG引擎) → PR3 (Agent升级) → PR4 (API+前端)
                      ↓
              PR1.5 必须先做，因为后续所有 PR 都需要持久化
```

---

## 二、PR1.5 — SQLite 持久化

### 2.1 新建文件

```
internal/repo/
  sqlite.go          ← OpenSQLite(dbPath) (*DB, error) + RunMigrations()
  migrations/
    001_init.sql     ← 现有所有表的 SQLite 版建表语句
```

### 2.2 container.go 改动

```go
// 现有:
if cfg.DatabaseURL != "" {
    // PostgreSQL
}

// 改为:
if cfg.DatabaseURL != "" {
    // PostgreSQL
} else {
    // SQLite 默认 ~/.dramora/data.db
    dbPath := cfg.DataDir + "/data.db"
    openedDB, err := repo.OpenSQLite(ctx, dbPath)
    projectRepo = repo.NewSQLiteProjectRepository(openedDB.Pool)
    productionRepo = repo.NewSQLiteProductionRepository(openedDB.Pool)
}
```

### 2.3 SQLite 建表（核心表）

```sql
-- 001_init.sql
CREATE TABLE IF NOT EXISTS projects (id TEXT PRIMARY KEY, ...);
CREATE TABLE IF NOT EXISTS episodes (id TEXT PRIMARY KEY, ...);
CREATE TABLE IF NOT EXISTS story_sources (...);
CREATE TABLE IF NOT EXISTS story_analyses (...);
CREATE TABLE IF NOT EXISTS generation_jobs (...);
CREATE TABLE IF NOT EXISTS approval_gates (...);
CREATE TABLE IF NOT EXISTS assets (...);
CREATE TABLE IF NOT EXISTS storyboard_shots (...);
CREATE TABLE IF NOT EXISTS timelines (...);
CREATE TABLE IF NOT EXISTS exports (...);

-- Phase 1 新增表
-- 002_phase1.sql
CREATE TABLE IF NOT EXISTS agent_runs (
    id TEXT PRIMARY KEY,
    analysis_id TEXT NOT NULL REFERENCES story_analyses(id),
    agent_role TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'waiting',  -- waiting|running|succeeded|failed
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    token_count INTEGER,
    model_name TEXT,
    output_description TEXT,
    output_highlights TEXT,         -- JSON array
    output TEXT,                    -- JSON object (结构化产出)
    raw_output TEXT,                -- LLM 原始响应
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS provider_configs (
    id TEXT PRIMARY KEY,
    capability TEXT NOT NULL UNIQUE,  -- chat|image|video|audio
    base_url TEXT NOT NULL,
    api_key TEXT NOT NULL,
    model TEXT NOT NULL,
    credits_per_unit INTEGER NOT NULL DEFAULT 0,
    credit_unit TEXT NOT NULL DEFAULT 'per_call',  -- per_call|per_second|per_char
    timeout_ms INTEGER DEFAULT 120000,
    max_retries INTEGER DEFAULT 3,
    is_enabled BOOLEAN DEFAULT 1,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_by TEXT
);

CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL DEFAULT 'creator',  -- admin|creator|viewer
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 积分系统表（Phase 3 完整启用，Phase 1 建好）
CREATE TABLE IF NOT EXISTS credit_wallets (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    balance INTEGER NOT NULL DEFAULT 0,
    frozen INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS credit_transactions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    amount INTEGER NOT NULL,
    type TEXT NOT NULL,               -- charge|consume|refund|expire|shadow
    capability TEXT,                   -- chat|image|video|audio
    project_id TEXT,
    episode_id TEXT,
    shot_id TEXT,
    model_name TEXT,
    actual_cost_cny REAL,
    credit_cost INTEGER,
    metadata TEXT,                     -- JSON
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

---

## 三、PR1 — 端点适配器

### 3.1 Go 接口

```go
// internal/provider/adapter.go

type ChatProviderConfig struct {
    BaseURL        string
    APIKey         string
    Model          string
    CreditsPerCall int
    Timeout        time.Duration
    MaxRetries     int
}
// ImageProviderConfig, VideoProviderConfig, AudioProviderConfig 同理

type ProviderSet struct {
    Chat  ChatProviderConfig
    Image ImageProviderConfig
    Video VideoProviderConfig
    Audio AudioProviderConfig
}
```

### 3.2 管理后台 API

```
GET  /api/v1/admin/providers          → 返回当前 4 个能力的配置 (api_key 脱敏)
POST /api/v1/admin/providers/:test    → 测试连接 (发一个简单请求验证端点可用)
POST /api/v1/admin/providers/:save    → 保存配置

请求体 (POST save):
{
  "capability": "chat",
  "base_url": "https://your-gateway.com/v1",
  "api_key": "sk-xxx",
  "model": "deepseek-chat",
  "credits_per_unit": 5,
  "credit_unit": "per_call"
}
```

### 3.3 错误场景

| 场景 | HTTP | 响应 |
|------|------|------|
| 某能力未配置，Agent 尝试调用 | 503 | `{"error": "chat 端点未配置，请在管理后台配置"}` |
| api_key 无效 | 502 | `{"error": "端点返回 401，请检查 API Key"}` |
| 端点超时 | 504 | `{"error": "端点超时 (120s)，请检查网络或延长超时"}` |
| 测试连接成功 | 200 | `{"ok": true, "model": "deepseek-chat", "latency_ms": 320}` |

---

## 四、PR2 — DAG 引擎

### 4.1 核心类型

```go
// internal/workflow/engine.go

type Engine struct {
    graph      *Graph
    blackboard *Blackboard
    executor   AgentExecutor  // 调用 UnifiedAdapter
}

type NodeRun struct {
    ID       string
    NodeID   string
    Kind     NodeKind
    Status   NodeRunStatus  // waiting|running|succeeded|failed|skipped
    AgentRun *AgentRun      // 关联的 Agent 执行记录
}

type NodeRunStatus string
const (
    NodeWaiting   NodeRunStatus = "waiting"
    NodeRunning   NodeRunStatus = "running"
    NodeSucceeded NodeRunStatus = "succeeded"
    NodeFailed    NodeRunStatus = "failed"
    NodeSkipped   NodeRunStatus = "skipped"
)

// Execute 执行 DAG，按拓扑顺序推进
func (e *Engine) Execute(ctx context.Context, workflowRunID string) error
```

### 4.2 Blackboard

```go
// internal/workflow/blackboard.go

type Blackboard struct {
    mu     sync.RWMutex
    state  map[string]interface{}  // role → output
    subscribers map[string][]chan BlackboardEvent
}

func (b *Blackboard) Write(role string, output interface{})
func (b *Blackboard) Read(role string) (interface{}, bool)
func (b *Blackboard) Subscribe(role string) <-chan BlackboardEvent
```

### 4.3 Phase 1 的 DAG 拓扑

```go
var Phase1Graph = &Graph{
    Nodes: []Node{
        {ID: "story_analyst",    Kind: NodeKindStoryAnalysis},
        {ID: "outline_planner",  Kind: NodeKindStoryAnalysis},
        {ID: "character_analyst", Kind: NodeKindCharacterDesign},
        {ID: "scene_analyst",    Kind: NodeKindSceneDesign},
        {ID: "prop_analyst",     Kind: NodeKindPropDesign},
    },
    Edges: []Edge{
        {From: "story_analyst",    To: "outline_planner"},
        {From: "outline_planner",  To: "character_analyst"},
        {From: "outline_planner",  To: "scene_analyst"},
        {From: "outline_planner",  To: "prop_analyst"},
    },
}
// character/scene/prop 三个无依赖关系 → 可并行执行
```

---

## 五、PR3 — Agent 升级

### 5.1 Prompt 模板（每个 Agent 一个）

```go
// internal/service/prompts.go

const StoryAnalystPrompt = `你是一个专业的故事分析师。请分析以下小说片段：

{source_text}

请输出 JSON：
{
  "themes": ["主题1", "主题2"],
  "conflict": "核心冲突描述",
  "main_plot": "故事主线概述（50字内）"
}`

const OutlinePlannerPrompt = `你是一个大纲规划师。请将以下故事拆分为四个情节点：

故事主线：{main_plot}

请输出 JSON：
{
  "beats": [
    {"code": "B01", "title": "开端", "summary": "...", "visual_goal": "..."},
    {"code": "B02", "title": "发展", "summary": "...", "visual_goal": "..."},
    {"code": "B03", "title": "转折", "summary": "...", "visual_goal": "..."},
    {"code": "B04", "title": "高潮", "summary": "...", "visual_goal": "..."}
  ]
}`

// CharacterAnalystPrompt, SceneAnalystPrompt, PropAnalystPrompt 同理
```

### 5.2 agent_service.go

```go
// internal/service/agent_service.go

type AgentService struct {
    adapter  *provider.UnifiedAdapter
    repo     AgentRunRepository
    prompts  *PromptRegistry
}

func (s *AgentService) ExecuteAgent(
    ctx context.Context,
    agentRole string,
    input AgentInput,
) (*AgentRun, error) {
    // 1. 创建 AgentRun 记录 (status=running)
    // 2. 获取 Prompt 模板
    // 3. 调用 UnifiedAdapter.ChatCompletion()
    // 4. 解析 JSON 响应
    // 5. 更新 AgentRun (status=succeeded, output=parsed)
    // 6. 写入 Blackboard
    // 失败 → 更新 AgentRun (status=failed, error_message=...)
}
```

### 5.3 story_analyzer.go 改动

```go
// 旧代码：analyzeStorySource() → 确定性正则/关键词
// 新代码：走 AgentService.ExecuteAgent() → LLM

// 旧函数保留，标记 Deprecated，作为无 LLM 端点时的 fallback
// Deprecated: 使用 AgentService.ExecuteAgent 代替
func analyzeStorySource(source domain.StorySource) deterministicStoryAnalysis {
    // 保留旧代码不动
}
```

---

## 六、PR4 — API + 前端

### 6.1 新增后端路由

```go
// GET  /api/v1/story-analyses/{analysisId}          ← 已有，扩展返回 agent_runs
// GET  /api/v1/story-analyses/{analysisId}/agent-runs ← 新增
// GET  /api/v1/admin/providers                       ← 新增 (管理后台)
// POST /api/v1/admin/providers:test                  ← 新增
// POST /api/v1/admin/providers:save                  ← 新增
// POST /api/v1/auth/register                         ← 新增 (Phase 2 启用)
// POST /api/v1/auth/login                            ← 新增 (Phase 2 启用)
// GET  /api/v1/auth/me                               ← 新增 (Phase 2 启用)
```

### 6.2 前端新增文件

```
apps/studio/src/
  studio/components/agent/
    AgentBoard.tsx           ← 7 个新组件
    AgentCard.tsx
    AgentPipeline.tsx
    AgentOutputPanel.tsx
    BlackboardView.tsx
    GlobalAgentIndicator.tsx
    index.ts
  studio/pages/
    AdminSettingsPage.tsx    ← 新增管理后台页面
  api/
    types.ts                 ← 扩展 AgentRunDTO, ProviderConfigDTO
    client.ts                ← 扩展 admin API 函数
    hooks.ts                 ← 扩展 useAgentRuns, useProviderConfigs
  studio/
    utils.ts                 ← 扩展 agentRoleLabel, formatTokens
    routes.ts                ← 扩展 /admin/settings 路由
  index.css                  ← 追加 200+ 行 agent 相关 CSS
```

### 6.3 前端组件 Props 契约

```typescript
// AgentBoard
interface AgentBoardProps {
  agents: AgentRunDTO[];
  onSelectAgent: (agent: AgentRunDTO) => void;
  expandedAgentId?: string;
}

// AgentCard
interface AgentCardProps {
  agent: AgentRunDTO;
  expanded: boolean;
  onSelect: () => void;
}

// AgentPipeline
interface AgentPipelineProps {
  agents: AgentRunDTO[];
  onNodeClick: (agent: AgentRunDTO) => void;
}

// AgentOutputPanel
interface AgentOutputPanelProps {
  agent: AgentRunDTO;
  onClose: () => void;
}

// GlobalAgentIndicator
interface GlobalAgentIndicatorProps {
  agents: AgentRunDTO[];
}
```

### 6.4 CSS 追加清单

在 `index.css` 末尾追加 §1.3 的全部 CSS 类（约 200 行），包括：
- `.agent-board`, `.agent-card`, `.agent-card.running/.succeeded/.failed/.expanded`
- `.status-dot.agent-running/.agent-waiting/.agent-done/.agent-failed`
- `@keyframes agent-pulse`, `@keyframes stream-blink`
- `.agent-streaming`, `.agent-error`, `.agent-output-detail`, `.agent-output-tag`
- `.agent-pipeline`, `.pipeline-node`, `.pipeline-edge`, `.pipeline-legend`
- `.agent-output-panel`, `.output-tabs`, `.beat-card`
- `.character-tag`, `.scene-tag`, `.prop-tag`
- `.blackboard-view`, `.blackboard-grid`, `.blackboard-section`
- `.global-agent-indicator`
- `@media (prefers-reduced-motion: reduce)` 动画禁用

---

## 七、验收 Checklist

### PR1.5 (SQLite)
- [ ] 不配 DATABASE_URL 时自动创建 SQLite 数据库
- [ ] 启动时自动建表（migrations 执行）
- [ ] 现有所有 API 在 SQLite 下正常工作
- [ ] WAL 模式已启用
- [ ] DATABASE_URL 配置后自动切 PostgreSQL（不破坏现有数据）

### PR1 (端点适配器)
- [ ] 管理后台可配置 4 个端点
- [ ] "测试连接"可验证端点可用性
- [ ] api_key 脱敏显示（只显示前 4 后 4 位）
- [ ] 未配置端点时返回明确错误
- [ ] 端点配置存 DB，重启不丢失

### PR2 (DAG 引擎)
- [ ] 5 节点 DAG 拓扑正确执行
- [ ] character/scene/prop 三个 Agent 可并行
- [ ] Blackboard 读写正常
- [ ] 节点失败不阻塞无依赖的兄弟节点

### PR3 (Agent 升级)
- [ ] 5 个 Agent 全部走 LLM 端点
- [ ] AgentRun 记录正确持久化
- [ ] 旧确定性函数保留但不调用

### PR4 (API + 前端)
- [ ] AgentBoard 展示 5 个 Agent 状态
- [ ] AgentCard 4 种状态正确渲染
- [ ] AgentPipeline SVG DAG 图正确
- [ ] AgentOutputPanel 按 Agent 类型适配展示
- [ ] GlobalAgentIndicator 在 board-header 显示
- [ ] 2 秒轮询 running Agent
- [ ] Go test + Studio lint/build 通过

---

## 八、总代码量估算

| PR | 新增文件 | 新增行数 | 修改文件 | 修改行数 |
|----|---------|---------|---------|---------|
| PR1.5 SQLite | 3 | ~200 | 1 | ~20 |
| PR1 端点 | 5 | ~400 | 1 | ~10 |
| PR2 DAG | 3 | ~300 | 0 | ~0 |
| PR3 Agent | 3 | ~500 | 1 | ~30 |
| PR4 API+前端 | 9 | ~900 | 6 | ~100 |
| **合计** | **23** | **~2300** | **9** | **~160** |
