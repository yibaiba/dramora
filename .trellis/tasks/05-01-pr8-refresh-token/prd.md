# PR8 — Refresh Token MVP

## Goal

把目前"7 天长 access token + 单 token 模型"升级为"短 access token (1h) + 长 refresh token (30d, 旋转) + 主动 logout"的标准会话模型，
让 token 即便泄露也只暴露 1h 的窗口，同时保留 30 天免登录体验。

## Decisions

- access token TTL = 1h（业界惯例）
- refresh token TTL = 30d
- 旋转：每次 refresh 产出新 refresh token，旧的立即吊销
- 新增 `POST /api/v1/auth/logout` 主动吊销当前 refresh token
- 前端 401 时自动用 refresh token 兑换新 access，再重放原请求

## Out of scope

- 多设备会话列表 / 单点下线（留给独立任务）
- refresh token 重用检测（reuse detection / revocation chain）
- audit log
- OAuth / SSO

## Backend changes

1. **DB schema**
   - 新增 `auth_refresh_tokens` 表：`id (uuid, pk)`, `user_id (uuid, fk users.id)`, `organization_id (uuid)`, `role (text)`, `token_hash (text, sha256 hex)`, `created_at`, `expires_at`, `revoked_at (nullable)`, `replaced_by_id (nullable)`。
   - PG migration `000015_create_auth_refresh_tokens.up/down.sql`；SQLite 启动 schema 同步。
   - `token_hash` 为 sha256(raw token)，不存明文。

2. **Repo 层**
   - 新增 `AuthRefreshTokenRepository`（Memory / SQLite / Postgres）方法：
     - `Create(ctx, params) (RefreshTokenRecord, error)`
     - `GetByHash(ctx, hash) (RefreshTokenRecord, error)`
     - `Revoke(ctx, id, replacedBy *string) error`
     - `RevokeAllForUser(ctx, userID) error`（logout-all 留 hook，本次不暴露）

3. **Service 层**
   - `AuthService.tokenTTL` 缩短到 1h；新增 `refreshTTL = 30 * 24h`。
   - `buildSession` 改为返回 `(AuthSession, refreshTokenPlain string, error)`：在签发 access JWT 同时生成 raw refresh token (32 bytes base64)，落库 hash。
   - 新增 `Refresh(ctx, refreshToken string) (AuthSession, refreshTokenPlain, error)`：
     - 计算 hash → `GetByHash` → 校验 `revoked_at IS NULL` 且 `expires_at > now`
     - 调用 `Revoke(old, replacedBy=newID)` → 签发新 access + 新 refresh
     - 失败统一返回 `ErrUnauthorized`
   - 新增 `Logout(ctx, refreshToken string) error`：`Revoke(found, nil)`；token 不存在或已吊销时静默返回 nil（避免泄露存在性）。
   - `Register / Login / RestoreSession` 三处统一使用新的 `buildSession`。

4. **HTTP 层**
   - `AuthSession` HTTP DTO 增加 `refresh_token` 与 `refresh_expires_at` 字段。
   - 新增 `POST /api/v1/auth/refresh`（公开路由，不需要 bearer），body `{ "refresh_token": "..." }`。
   - 新增 `POST /api/v1/auth/logout`（公开路由），body `{ "refresh_token": "..." }`，永远返回 204。
   - `api/openapi.yaml` 同步声明两个新路由 + DTO 字段。

5. **测试**
   - service：refresh 成功旋转、refresh 旧 token 二次使用失败、refresh 已过期失败、logout 后再 refresh 失败。
   - HTTP：register/login 返回 refresh_token；refresh 接口正常工作；logout 后 refresh 返回 401。

## Frontend changes

1. **存储**
   - `authStore` 新增 `refreshToken` 字段，与 `token` 一起持久化到 `localStorage`。
   - 注册 / 登录 / refresh 成功后同步更新两者；logout / 401 终态时清空两者。

2. **API client**
   - `api/client.ts` 加一层"自动 refresh 拦截"：当响应 401 且当前请求不是 `/auth/refresh|login|register|logout` 时，调用 `/auth/refresh` 兑换新 token，成功后重放一次原请求；refresh 也失败则清空 store 让 AuthPage 接管。
   - 同一时间多个并发 401 共享同一次 refresh promise，避免 thundering herd。

3. **Logout 入口**
   - StudioShell 退出按钮改为先调用 `/auth/logout` 再清 store（失败也清）。

## Acceptance Criteria

- [x] 访问 `/api/v1/auth/login` 返回 `token` (1h) + `refresh_token` (30d) + `refresh_expires_at`。
- [x] `POST /api/v1/auth/refresh` 成功后旧 refresh_token 立即不可再用（DB `revoked_at` 非空、`replaced_by_id` 指向新 token）。
- [x] `POST /api/v1/auth/logout` 后再用同一 refresh_token 调 `/refresh` 返回 401。
- [x] 前端 access token 过期时自动 refresh，业务请求对调用方透明（不弹回登录页）。
- [x] refresh token 不存明文：DB 字段是 sha256 hex。
- [x] `cd apps/studio && npm run lint && npm run build` 0 errors / warnings ≤16。
- [x] `GOTOOLCHAIN=local go test ./...`、`GOTOOLCHAIN=local go build ./...` 全绿。

## DoD

- 上面 acceptance 全部满足
- OpenAPI 同步
- 任务收尾时归档到 archive

## Technical Notes

- refresh token 生成：`crypto/rand` 32 bytes → `base64.RawURLEncoding`。
- hash：`sha256.Sum256(raw)` → hex；查询时同样 hash 后等值比较。
- 不在 JWT 内嵌 refresh：refresh 是独立的 opaque token，便于轮换/吊销。
- 旋转时使用单事务（PG）/串行更新（SQLite/Memory）保证旧 token revoke 与新 token insert 原子。
