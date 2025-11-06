# Intercept Wave Upstream — Upstream Test Orchestrator

This repository provides the official upstream testing services for
`Intercept Wave` — used for manual verification and CI automation of
forwarding, matching and WebSocket features.

Container image: `ghcr.io/zhongmiao-org/intercept-wave-upstream`

- 3 HTTP services on ports 9000, 9001, 9002
- 3 WebSocket services on ports 9003, 9004, 9005
- Rich test endpoints to validate Intercept Wave features: prefix variations, method echo, headers/cookies, delay, status, large payloads, wildcard-like paths, and WebSocket echo/ticker/timeline.

## Run

## Docker

- Build: \`docker build -t intercept-wave-upstream .\`
- Run: \`docker run --rm -p 9000-9005:9000-9005 intercept-wave-upstream\`
- Compose: \`docker compose up -d\`

Release pipeline builds multi-arch images and pushes to GHCR on GitHub Releases.

```
go run ./...
```

By default, it starts 6 servers:
- HTTP: 9000 (user), 9001 (order), 9002 (payment)
- WS:   9003, 9004, 9005

Environment overrides:
- `BASE_PORT` (default `9000`): HTTP uses BASE_PORT..BASE_PORT+2, WS uses BASE_PORT+3..BASE_PORT+5

## Example HTTP APIs

- `GET /` — service info
- `GET /health` — health check
- `GET /status/{code}` — return a specific HTTP status
- `GET /delay/{ms}` — respond after ms delay
- `POST/PUT/PATCH /echo` — echo request body
- `GET /headers` — return selected headers
- `GET /cookies` — return request cookies
- `GET /large?size=65536` — large JSON payload

Service-specific examples:
- 9000 User: `GET /api/user/info`, `GET /api/posts`
- 9001 Order: `GET /order-api/orders`, `POST /order-api/orders`
- 9002 Payment: `POST /pay-api/checkout`

## Example WebSocket APIs

Connect to:
- `ws://localhost:9003/ws/echo` — echo messages
- `ws://localhost:9004/ws/ticker?interval=1000` — periodic messages
- `ws://localhost:9005/ws/timeline` — fixed sequence then close

## How it pairs with Intercept Wave

- Point the Intercept Wave proxy groups to these services:
  - Group "User" → `baseUrl=http://localhost:9000`, `interceptPrefix=/api`, `stripPrefix=true`
  - Group "Order" → `baseUrl=http://localhost:9001`, `interceptPrefix=/order-api`, `stripPrefix=true`
  - Group "Payment" → `baseUrl=http://localhost:9002`, `interceptPrefix=/pay-api`, `stripPrefix=true`
- Use provided endpoints to validate:
  - Forwarding preserves headers/body/status
  - CORS behavior (plugin sets headers) and delays
  - Wildcard-like paths with `stripPrefix`

## CI & Release

- CI (PR/main): build, vet, test, fmt check.
- Release draft (push to main): created from the `VERSION` file.
- Publish release: builds multi-arch images and pushes to GHCR.

Repository configuration:
- Secrets (optional)
  - `GHCR_TOKEN` — a PAT with `write:packages` (only if the repository owner cannot push to `zhongmiao-org` with the default `GITHUB_TOKEN`)
- Workflow permissions
  - Settings → Actions → General → Workflow permissions: enable "Read and write permissions"
  - Workflow sets `packages: write` to push to GHCR

Notes:
- If this repository belongs to the `zhongmiao-org` organization, no extra token is required — `GITHUB_TOKEN` can push to `ghcr.io/zhongmiao-org/...`.
- If the repository is outside `zhongmiao-org` but you still want to push to that namespace, add `GHCR_TOKEN` (PAT with `write:packages` and SSO enabled for the org).
