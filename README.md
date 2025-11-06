<div align="center">

<img src="docs/logo.svg" alt="Intercept Wave Upstream" height="88" width="88" />

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/plus-dark.svg" />
  <source media="(prefers-color-scheme: light)" srcset="docs/plus-light.svg" />
  <img src="docs/plus-light.svg" alt="+" height="88" />
</picture>

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://go.dev/images/go-logo-white.svg" />
  <source media="(prefers-color-scheme: light)" srcset="https://go.dev/images/go-logo-blue.svg" />
  <img src="https://go.dev/images/go-logo-blue.svg" alt="Go" height="88" />
</picture>

# Intercept Wave Upstream — Upstream Test Orchestrator


[![CI](https://img.shields.io/github/actions/workflow/status/zhongmiao-org/intercept-wave-upstream/ci.yml?branch=main&label=CI&style=flat-square)](https://github.com/zhongmiao-org/intercept-wave-upstream/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/actions/workflow/status/zhongmiao-org/intercept-wave-upstream/release.yml?label=Release&style=flat-square)](https://github.com/zhongmiao-org/intercept-wave-upstream/actions/workflows/release.yml)
[![GitHub release](https://img.shields.io/github/v/release/zhongmiao-org/intercept-wave-upstream?sort=semver&display_name=tag&style=flat-square)](https://github.com/zhongmiao-org/intercept-wave-upstream/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&style=flat-square)](https://go.dev/)
[![GHCR](https://img.shields.io/badge/GHCR-intercept--wave--upstream-2ea44f?logo=github&style=flat-square)](https://github.com/zhongmiao-org/intercept-wave-upstream/pkgs/container/intercept-wave-upstream)
[![Platforms](https://img.shields.io/badge/Platforms-amd64%20%7C%20arm64-6aa84f?style=flat-square)](#)
[![Code Style](https://img.shields.io/badge/code%20style-gofmt-FFD54F?style=flat-square)](#)

[English](README.md) | 简体中文

</div>

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

## HTTP Endpoints and Examples

Base endpoints (all HTTP services expose these):

- GET /
  - Response:
    ```json
    {
      "service": "user-service",
      "port": 9000,
      "interceptPrefix": "/api",
      "message": "Upstream running"
    }
    ```
- GET /health
  - Response: `{"status":"ok"}`
- GET /status/418
  - Response: `{"status":418}` (status code: 418)
- GET /delay/500
  - Response after ~500ms: `{"delayedMs":500}`
- POST /echo (also supports PUT/PATCH)
  - Request: `curl -X POST http://localhost:9000/echo -d '{"name":"abc"}' -H 'Content-Type: application/json'`
  - Response:
    ```json
    {
      "method": "POST",
      "path": "/echo",
      "query": "",
      "length": 13,
      "body": "{\"name\":\"abc\"}"
    }
    ```
- GET /headers (echo selected request headers)
  - Request: `curl -H 'Authorization: Bearer 123' http://localhost:9001/headers`
  - Response: `{"headers":{"Authorization":"Bearer 123"}}`
- GET /cookies (echo request cookies)
  - Request: `curl -H 'Cookie: sid=abc; user=tom' http://localhost:9002/cookies`
  - Response: `{"cookies":{"sid":"abc","user":"tom"}}`
- GET /large?size=64
  - Response (truncated):
    ```json
    { "size": 64, "data": "aaaaaaaaaaaaaaaa..." }
    ```

Service-specific endpoints:

1) User service (9000, interceptPrefix=/api)

- GET /api/user/info
  - Response:
    ```json
    {
      "code": 0,
      "data": {
        "id": 1,
        "name": "张三",
        "email": "zhangsan@example.com"
      },
      "message": "success"
    }
    ```
- GET /api/posts
  - Response (example):
    ```json
    {
      "code": 0,
      "data": [
        { "id": 1, "title": "Post 1", "createdAt": "2024-01-01T12:00:00Z" },
        { "id": 2, "title": "Post 2", "createdAt": "2024-01-01T11:00:00Z" }
      ]
    }
    ```

2) Order service (9001, interceptPrefix=/order-api)

- GET /order-api/orders
  - Response:
    ```json
    {
      "code": 0,
      "data": [
        { "id": 1001, "status": "CREATED" },
        { "id": 1002, "status": "PAID" }
      ]
    }
    ```
- POST /order-api/orders
  - Request: `curl -X POST http://localhost:9001/order-api/orders -H 'Content-Type: application/json' -d '{"sku":"123","qty":2}'`
  - Response (201 Created, server adds id):
    ```json
    {
      "code": 0,
      "data": { "sku": "123", "qty": 2, "id": 54321 }
    }
    ```
- GET /order-api/order/123/submit
  - Response: `{"message":"submit ok"}`

3) Payment service (9002, interceptPrefix=/pay-api)

- POST /pay-api/checkout
  - Response (simulated ~150ms delay):
    ```json
    {
      "code": 0,
      "data": { "paid": true, "amount": 199, "currency": "CNY" },
      "message": "paid"
    }
    ```

## WebSocket Endpoints and Examples

1) Echo (9003)
- Connect: `ws://localhost:9003/ws/echo`
- Behavior: server echoes any text/binary frames
- Example: send `ping` → receive `ping`

2) Ticker (9004)
- Connect: `ws://localhost:9004/ws/ticker?interval=1000`
- Behavior: server pushes `tick 1`, `tick 2`, ... every `interval` ms (default 1000)

3) Timeline (9005)
- Connect: `ws://localhost:9005/ws/timeline`
- Behavior: server sends `hello`, `processing`, `done`, then closes with normal closure (reason: `bye`)

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
- Release draft (push to main): created using `CHANGELOG.md` Unreleased notes and the `VERSION` file.
- Publish release: builds multi-arch images and pushes to GHCR. CI promotes `Unreleased` notes into a new versioned section and opens a PR updating `CHANGELOG.md` on `main`.

Repository configuration:
- Secrets (optional)
  - `GHCR_TOKEN` — a PAT with `write:packages` (only if the repository owner cannot push to `zhongmiao-org` with the default `GITHUB_TOKEN`)
- Workflow permissions
  - Settings → Actions → General → Workflow permissions: enable "Read and write permissions"
  - Workflow sets `packages: write` to push to GHCR

Notes:
- If this repository belongs to the `zhongmiao-org` organization, no extra token is required — `GITHUB_TOKEN` can push to `ghcr.io/zhongmiao-org/...`.
- If the repository is outside `zhongmiao-org` but you still want to push to that namespace, add `GHCR_TOKEN` (PAT with `write:packages` and SSO enabled for the org).
