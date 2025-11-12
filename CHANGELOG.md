# Changelog

All notable changes to this project will be documented in this file.

The format is inspired by Keep a Changelog, and this project adheres to Semantic Versioning.

## [Unreleased]

- No changes yet.

## [0.2.0] - 2025-11-12


### Added
- WS: food-delivery simulation endpoints on all WS services
  - `ws://.../ws/food/user` and `ws://.../ws/food/merchant`
  - Diverse event key per service: `type` (ws-echo), `action` (ws-ticker), `event` (ws-timeline)
  - Supports `?interval=` throttling and asset-driven payloads with fallbacks
- HTTP: RESTful demo endpoints available on all HTTP services
  - `GET|POST|OPTIONS /rest/items`
  - `GET|PUT|PATCH|DELETE|OPTIONS /rest/items/{id}` with in-memory store
- Data: static JSON assets directory with overrideable `ASSETS_DIR`
  - Endpoints can load from `assets/` (e.g., `user/info.json`, `user/posts.json`, `order/orders.json`, `payment/checkout.json`, `ws/timeline.json`, `ws/food_*.json`, `rest/items.json`)
- Docs: comprehensive Chinese API list at `docs/接口清单.md`
  - Full ports and paths, WS token usage, REST endpoints, asset sources, and WS event-key rules

### Changed
- WS security: enforce a hardcoded token for all WS endpoints
  - Accept `X-Auth-Token: zhongmiao-org-token` header or `?token=zhongmiao-org-token` query param
- HTTP/WS behavior: several responses now read from static assets when available, with sensible built-in defaults

### Fixed
- Docs: normalize JSON examples in `docs/接口清单.md` to valid JSON (arrays/objects) and remove ambiguous placeholders


## [0.1.1] - 2025-11-06


### Changed
- CI: releaseDraft now checks all commits in a push to `main` for `VERSION`/`CHANGELOG.md` changes (no longer limited to `head_commit`).
- CI: unify `release.yml` into a single job that builds images from the release tag and then opens a PR to update `CHANGELOG.md` on `main`.
- CI: post-release flow no longer bumps `VERSION`; only `CHANGELOG.md` is updated.
- Docs: unify README headers (centered block with Go + project logo), enlarge header icons for better balance, and align Chinese badges with English.

### Added
- Docs: add `README_zh.md` and language switch link in `README.md`.
- Docs: clarify release process and GHCR publishing notes.
- Assets: add theme-aware plus icons `docs/plus-light.svg` and `docs/plus-dark.svg` and use them between logos in README headers.
- Docs: switch to SVG project logo (`docs/logo.svg`) in both READMEs for crisp rendering.


## [0.1.0] - 2025-11-06


### Added
- Go 1.25 module that boots a lightweight upstream test orchestrator for Intercept Wave.
- 3 HTTP services on consecutive ports (BASE_PORT..BASE_PORT+2, default 9000–9002), each providing shared utility endpoints:
  - `GET /` – service info (includes `interceptPrefix`)
  - `GET /health`
  - `GET /status/{code}`
  - `GET /delay/{ms}`
  - `POST|PUT|PATCH /echo` – echoes method, path, query and body
  - `GET /headers` – echoes selected request headers
  - `GET /cookies` – echoes request cookies
  - `GET /large?size=` – large JSON payload (capped to 2MB)
- Service-specific mock APIs aligned with Intercept Wave samples:
  - user-service (port `BASE`, `interceptPrefix=/api`):
    - `GET /api/user/info`
    - `GET /api/posts`
  - order-service (port `BASE+1`, `interceptPrefix=/order-api`):
    - `GET /order-api/orders`
    - `POST /order-api/orders` (creates and returns an id)
    - `GET /order-api/order/{id}/submit` (wildcard-like path)
  - payment-service (port `BASE+2`, `interceptPrefix=/pay-api`):
    - `POST /pay-api/checkout` (simulated latency)
- 3 WebSocket services on ports BASE+3..BASE+5 (default 9003–9005):
  - `ws://.../ws/echo` – echoes text/binary frames
  - `ws://.../ws/ticker?interval=1000` – periodic messages (`tick N`)
  - `ws://.../ws/timeline` – sends `hello`, `processing`, `done`, then closes normally
- Environment variable `BASE_PORT` to shift both HTTP and WS port ranges.
- Docker multi-stage build producing a minimal static binary image (`scratch`) and a `.dockerignore`.
- `docker-compose.yml` exposing ports 9000–9005 with `BASE_PORT` configurable.
- CI workflow (`ci.yml`): build, vet, test, gofmt check; push to `main` creates a Release Draft from `CHANGELOG.md` Unreleased notes.
- Release workflow (`release.yml`): multi-arch (amd64/arm64) build and push to GHCR at `ghcr.io/zhongmiao-org/intercept-wave-upstream`.
- Post-release automation: promote `Unreleased` into the release section, ensure `CHANGELOG.md` has a fresh `Unreleased` section, and open an auto-merge PR.
- Documentation:
  - README with container usage, detailed endpoint examples and pairing guidance with Intercept Wave.
  - CHANGELOG initialized with Unreleased.
- Unit tests:
  - HTTP: service boot and `GET /order-api/orders` returns `{ code: 0, ... }`.
  - WS: echo server connects and echoes payload.
