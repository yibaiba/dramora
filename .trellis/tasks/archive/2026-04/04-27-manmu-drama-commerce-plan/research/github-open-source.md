# GitHub Open Source Research: Manmu Drama + Commerce

## Goal

Research reference projects for a Go backend + React frontend product that combines short drama/comic content, ecommerce, and optional creator canvas features.

## Current repository context

- The current `dramora` repository is mostly Trellis scaffolding and does not yet contain product backend/frontend code.
- `.trellis/spec/backend/*` and `.trellis/spec/frontend/*` exist but are still template-level and mostly unfilled.
- This means the product architecture can be planned from a clean slate while still following Trellis task/PRD workflow.

## Reference projects

### Video / drama playback

**Owncast** — https://github.com/owncast/owncast

- Backend is Go; frontend is React.
- Focuses on self-hosted live video streaming and chat.
- Supports broadcast software via RTMP and runs a web interface/admin interface.
- Useful lessons for Manmu:
  - Keep media service boundaries explicit.
  - Use Go for long-running media/backend services.
  - Separate playback UI, admin UI, and backend service.
- MVP implication:
  - Do not start with live streaming unless required.
  - Start with VOD: upload, async transcode, object storage, HLS playback, CDN-ready URLs.

### Ecommerce / headless commerce

**Saleor** — https://github.com/saleor/saleor

- API-only, headless commerce platform.
- GraphQL-native and multichannel by design.
- Strong model for product/catalog/order/payment/channel separation.
- Useful lessons for Manmu:
  - Commerce should be API-first and decoupled from presentation.
  - Channels matter if Manmu later splits into `漫幕` and `电商漫幕`.
  - Extensions/webhooks are safer than tightly coupled plugins.

**Saleor Storefront** — https://github.com/saleor/storefront

- React/Next.js, TypeScript, GraphQL, Tailwind storefront.
- Useful as frontend reference for product listing, cart, checkout, and storefront composition.

**Medusa** — https://github.com/medusajs/medusa

- Commerce building blocks with customization framework and modules.
- Useful lessons for Manmu:
  - Treat commerce capabilities as composable primitives: products, variants, carts, orders, promotions, fulfillment, payments.
  - Avoid hardcoding all commerce logic into content pages.

### Go admin / backend scaffolding

**gin-vue-admin** — https://github.com/flipped-aurora/gin-vue-admin

- Go Gin backend with GORM, JWT, Casbin, Redis, Swagger, Zap, Viper.
- Frontend is Vue, so not directly adopted for Manmu frontend.
- Useful lessons for Manmu:
  - Backend layering: API, router, service, model, middleware, config, utils.
  - RBAC/Casbin-style permission model is suitable for creator/admin/merchant roles.
  - Swagger/OpenAPI can keep frontend-backend contracts explicit.

### Canvas / creator tooling

**tldraw** — https://github.com/tldraw/tldraw

- React SDK for infinite canvas apps.
- Supports multiplayer sync, custom shapes/tools/bindings/UI, image/video support, exports, and AI integrations.
- Good fit for a creator canvas prototype.
- Caution:
  - Production use requires reviewing/commercial licensing.

**Excalidraw** — https://github.com/excalidraw/excalidraw

- MIT-licensed hand-drawn-style infinite canvas.
- Supports dark mode, image support, shape libraries, export, undo/redo, zoom/pan, collaboration, E2EE, local-first app behavior.
- Good fit for storyboarding, mood boards, scene planning, and creator collaboration.

**xyflow / React Flow** — https://github.com/xyflow/xyflow

- React library for node-based UIs.
- Good fit for plot graphs, episode branching, workflow builders, creator pipelines, and campaign automation.

## UI/UX research notes

The `ui-ux-pro-max` design-system search suggested:

- Product pattern: immersive / interactive experience.
- Style: vibrant and block-based, suitable for entertainment and youth-focused consumer products.
- Visual direction:
  - Large sections and bold hierarchy.
  - Strong CTA color.
  - Content-first media grids.
  - Dark/immersive mode should be considered for drama playback.
- Key performance/accessibility rules:
  - Avoid high-resolution autoplay loops by default.
  - Reserve aspect ratio for posters/video cards to prevent layout shift.
  - Lazy-load below-fold images and media.
  - Dynamically import heavy canvas/editor modules.
  - Use accessible contrast, visible focus states, and reduced-motion support.

## Recommended MVP architecture

Use a modular monolith first:

- Backend: Go API service organized by domain modules.
- Frontend: React website, preferably Next.js for SEO/content discovery, or Vite React if SEO is explicitly not important.
- Database: PostgreSQL.
- Cache/session/rate-limit: Redis.
- Object storage: S3-compatible storage or cloud OSS/COS.
- Search: Meilisearch/Typesense first; Elasticsearch/OpenSearch later only if scale requires it.
- Async jobs: Go worker with Asynq or similar Redis-backed queue.
- Media pipeline: upload original media, enqueue FFmpeg transcode, output HLS, serve via CDN/object storage.
- Canvas: start as optional creator module; persist canvas documents as JSON plus exported previews.

## Suggested modules

- Identity and auth: user, creator, merchant, admin roles.
- Content: drama/comic works, episodes, scenes, tags, recommendations.
- Media: posters, videos, subtitles, transcode jobs, playback manifests.
- Commerce: products, SKU/variants, cart, order, payment, refund, fulfillment.
- Creator tools: canvas documents, storyboards, exported covers/posters.
- Admin: moderation, content review, order operations, merchant operations.
- Search/discovery: keyword search, filters, trending lists.

## Main tradeoffs

### REST + OpenAPI vs GraphQL

- REST + OpenAPI is simpler for Go MVP, easier to test and document.
- GraphQL is strong for flexible storefront queries and multichannel commerce, as demonstrated by Saleor.
- Recommendation: start REST + OpenAPI; revisit GraphQL/BFF only after frontend query complexity becomes painful.

### Modular monolith vs microservices

- Modular monolith is faster and safer for MVP.
- Microservices help only after media processing, commerce, recommendations, and collaboration each need independent scale.
- Recommendation: modular monolith + separate worker process now; split services later.

### Canvas choice

- tldraw: best developer experience and extensibility, but production licensing must be checked.
- Excalidraw: strong open-source option, MIT, good for hand-drawn storyboards.
- React Flow: best for graph/workflow/plot nodes, not free-form drawing.
- Recommendation:
  - MVP storyboard/moodboard: Excalidraw.
  - MVP plot graph/workflow: React Flow.
  - Advanced creator suite: evaluate tldraw license and SDK fit.

## Open decision

The most important product decision is whether the first MVP should prioritize:

1. Content platform first: drama/comic viewing, content CMS, basic creator upload.
2. Commerce platform first: products, cart, checkout, orders, admin.
3. Creator platform first: canvas/storyboard/poster tools, then attach content and products.
4. Balanced MVP: content + product detail + lightweight canvas, with checkout/payment mocked or deferred.
