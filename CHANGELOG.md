# Changelog

All notable changes to this project will be documented in this file.

The format is inspired by Keep a Changelog, and this project adheres to Semantic Versioning.

## [Unreleased]

- No changes yet.

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
- Post-release automation: compute next patch version, update `VERSION`, ensure `CHANGELOG.md` has an `Unreleased` section, open an auto-merge PR.
- Documentation:
  - README with container usage, detailed endpoint examples and pairing guidance with Intercept Wave.
  - CHANGELOG initialized with Unreleased.
- Unit tests:
  - HTTP: service boot and `GET /order-api/orders` returns `{ code: 0, ... }`.
  - WS: echo server connects and echoes payload.

