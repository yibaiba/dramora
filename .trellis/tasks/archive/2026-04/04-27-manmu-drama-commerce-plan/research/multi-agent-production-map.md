# Multi-Agent Production Map for Manmu

## Purpose

Design a multi-agent collaboration model for Manmu's AI manju production workflow.

The key idea: do not let one generic LLM produce the entire manju. Use specialized agents with explicit responsibilities, shared state, human approval points, budgets, and traceable outputs.

## Multi-agent framework references

| Reference | Useful idea for Manmu |
| --- | --- |
| MetaGPT | Role-based team + SOP. "Code = SOP(Team)" maps well to "Manju = SOP(Creative Team)". |
| CrewAI | Role-specific agents collaborate on multi-step workflows. Good conceptual model for story/visual/director/editor agents. |
| LangGraph | Long-running stateful workflows, durable execution, human-in-the-loop, memory, debugging/observability. Good orchestration model for production jobs. |
| AutoGen | Multi-agent conversations and agent UI references. Useful for director/reviewer debates and iterative refinement. |
| Flock | Visual workflow nodes, agent nodes, subgraphs, human nodes, MCP/tool nodes. Good UI reference if Manmu later exposes workflow editing. |
| Network-AI | Shared state, guardrails, budgets, and cross-framework coordination. Useful for cost control and safety constraints. |
| Dify / Flowise | Prompt IDE, model/provider management, workflow logs, visual flow editing. Useful for admin/prompt operations. |

## Agent team for Manmu MVP

### 1. Producer Agent

Mission: Control production scope, duration, budget, and deliverable type.

Inputs:

- project goal
- target duration
- style
- model budget
- episode count

Outputs:

- production plan
- generation budget
- phase checklist
- approval gates

Important controls:

- prevents over-generation
- caps expensive video calls
- decides when to ask human approval

### 2. Story Analyst Agent

Mission: Analyze story/novel/script into structured narrative data.

Outputs:

- story summary
- theme
- conflict
- emotional curve
- timeline
- character candidates
- scene candidates
- prop candidates

### 3. Screenwriter Agent

Mission: Convert story analysis into episode script and scene breakdown.

Outputs:

- episode outline
- scene list
- dialogue
- narration
- pacing
- shot intent

### 4. Character Designer Agent

Mission: Build consistent character cards and visual references.

Outputs:

- character cards
- relationship map
- wardrobe variants
- full-body prompt
- turnaround prompt
- expression pack prompt
- pose pack prompt

Human gate:

- user locks accepted character version before shot generation.

### 5. Scene Designer Agent

Mission: Build Scene Map and set design.

Outputs:

- scene cards
- location map
- time/weather/lighting/mood
- color palette
- concept art prompts
- background plate prompts
- continuity notes

Human gate:

- user locks accepted scene concept image / version.

### 6. Prop Designer Agent

Mission: Extract and design reusable props.

Outputs:

- prop cards
- prop reference prompts
- owner/scene associations
- continuity notes

### 7. Storyboard Agent

Mission: Convert script into shot cards.

Outputs:

- shot list
- shot duration
- camera angle
- camera motion
- action
- dialogue / subtitle lines
- referenced character ids
- referenced scene id
- referenced prop ids

### 8. Prompt Engineer Agent

Mission: Convert structured shot/asset data into provider-specific prompts.

Outputs:

- image prompt
- video prompt
- negative prompt
- model parameters
- reference asset pack

Key rule:

- prompts must be generated from locked character + scene + prop references.

### 9. Director Agent

Mission: Decide visual continuity and shot production route.

Outputs:

- start frame plan
- optional end frame plan
- keyframe route vs single image-to-video route
- regeneration recommendation
- shot priority

### 10. Cinematographer Agent

Mission: Improve camera language.

Outputs:

- shot size
- lens feel
- camera movement
- composition
- lighting direction

### 11. Voice & Subtitle Agent

Mission: Generate speech, voice direction, captions, and subtitle timing.

Outputs:

- TTS script
- character voice style
- subtitle segments
- caption style preset

### 12. Editor Agent

Mission: Transform generated clips into timeline decisions.

Outputs:

- timeline assembly
- clip order
- trim suggestions
- transition choices
- rough cut notes

### 13. Continuity Supervisor Agent

Mission: Detect inconsistencies across story, characters, scenes, props, and shots.

Checks:

- character appearance drift
- wardrobe mismatch
- scene/location mismatch
- missing prop continuity
- timeline/order errors
- dialogue contradiction

Outputs:

- continuity issues
- severity
- suggested fixes
- regenerate target

### 14. Safety & Copyright Agent

Mission: Screen prompts and outputs before model calls/export.

Checks:

- unsafe content
- copyrighted character/style risks
- celebrity likeness risks
- platform policy risks

Outputs:

- block / warn / allow decision
- redacted prompt suggestion

### 15. Cost Controller Agent

Mission: Track and control model spending.

Outputs:

- estimated cost before generation
- actual cost after generation
- budget remaining
- fallback model recommendation

## Recommended orchestration pattern

Use a stateful DAG, not free-form chat:

```text
Producer
  -> Story Analyst
  -> Screenwriter
  -> Character Designer -> Human Lock
  -> Scene Designer     -> Human Lock
  -> Prop Designer      -> Human Lock
  -> Storyboard Agent
  -> Prompt Engineer
  -> Safety Agent
  -> Director / Cinematographer
  -> Generation Jobs
  -> Continuity Supervisor
  -> Voice & Subtitle
  -> Editor Agent
  -> Export Job
  -> Final Review
```

## Shared state model

All agents read/write structured artifacts, not loose chat text.

```text
ProjectState
  ├── production_plan
  ├── story_analysis
  ├── world_bible
  ├── character_map
  ├── scene_map
  ├── prop_map
  ├── shot_list
  ├── prompt_pack
  ├── generation_jobs
  ├── asset_library
  ├── continuity_report
  ├── timeline
  └── export_report
```

## Human-in-the-loop gates

MVP should include explicit approval points:

1. approve story analysis / direction
2. lock character references
3. lock scene concept images
4. approve storyboard before expensive video generation
5. approve final timeline before export

## Backend implications

New entities:

- AgentRun
- AgentStep
- AgentArtifact
- AgentMessage
- ApprovalGate
- ReviewIssue
- CostLedger

Agent runs must record:

- agent name/version
- input artifact ids
- output artifact ids
- model provider/model
- prompt template version
- token/cost metadata
- status/error

## UI implications

Manmu Studio should expose an "Agent Board":

- Current phase
- Active agent
- Waiting approval gates
- Generated artifacts
- Cost estimate and actual spend
- Issues found by Continuity/Safety agents
- Retry/regenerate buttons per artifact

## MVP recommendation

Do not expose a fully user-editable multi-agent workflow builder in v1.

Instead:

- implement fixed production SOP internally,
- show agent progress and artifacts in UI,
- allow human approvals at key gates,
- persist every agent artifact for reproducibility.
