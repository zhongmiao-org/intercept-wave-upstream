<div align="center">

<img src="docs/logo.svg" alt="Intercept Wave Upstream" height="88" />
&nbsp;
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
&nbsp;

# Intercept Wave Upstream — 上游测试编排器


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


本仓库提供 Intercept Wave 的“上游测试服务”，用于手动验证与 CI 自动化测试：转发、匹配、以及 WebSocket 场景。

容器镜像：`ghcr.io/zhongmiao-org/intercept-wave-upstream`

- 3 个 HTTP 服务：端口 9000、9001、9002
- 3 个 WebSocket 服务：端口 9003、9004、9005
- 丰富的测试端点：状态码/延迟/回显/大包/路径前缀，以及 WS 的 echo/ticker/timeline

## 快速开始

### Docker

- 构建：`docker build -t intercept-wave-upstream .`
- 运行：`docker run --rm -p 9000-9005:9000-9005 intercept-wave-upstream`
- Compose：`docker compose up -d`

本地运行：

```
go run ./...
```

启动后默认提供 6 个服务：
- HTTP：9000（user）、9001（order）、9002（payment）
- WS：9003、9004、9005

环境变量：
- `BASE_PORT`（默认 `9000`）：HTTP 使用 `BASE_PORT..BASE_PORT+2`，WS 使用 `BASE_PORT+3..BASE_PORT+5`

## 示例 HTTP 接口

通用端点（所有 HTTP 服务均提供）：
- `GET /`：服务信息（含 `interceptPrefix`）
- `GET /health`：健康检查
- `GET /status/{code}`：返回指定状态码
- `GET /delay/{ms}`：延迟响应
- `POST|PUT|PATCH /echo`：回显请求方法/路径/查询/长度/Body
- `GET /headers`：回显部分请求头
- `GET /cookies`：回显 Cookie
- `GET /large?size=65536`：返回大 JSON 负载

服务特定示例：
- 用户服务（9000，`interceptPrefix=/api`）：`GET /api/user/info`、`GET /api/posts`
- 订单服务（9001，`interceptPrefix=/order-api`）：`GET /order-api/orders`、`POST /order-api/orders`、`GET /order-api/order/{id}/submit`
- 支付服务（9002，`interceptPrefix=/pay-api`）：`POST /pay-api/checkout`

## 示例 WebSocket 接口

- Echo（9003）：`ws://localhost:9003/ws/echo`（回显文本/二进制帧）
- Ticker（9004）：`ws://localhost:9004/ws/ticker?interval=1000`（周期推送 `tick N`）
- Timeline（9005）：`ws://localhost:9005/ws/timeline`（依次发送 `hello`、`processing`、`done` 后正常关闭）

## 与 Intercept Wave 配合

在 Intercept Wave 中配置代理分组：
- User → `baseUrl=http://localhost:9000`，`interceptPrefix=/api`，`stripPrefix=true`
- Order → `baseUrl=http://localhost:9001`，`interceptPrefix=/order-api`，`stripPrefix=true`
- Payment → `baseUrl=http://localhost:9002`，`interceptPrefix=/pay-api`，`stripPrefix=true`

## CI 与发布

- CI（PR/main）：构建、`go vet`、测试、`gofmt` 检查
- Release Draft（推送到 main）：从 `CHANGELOG.md` 的 `Unreleased` 段生成草稿发布
- 发布 Release：在发布/预发布时，基于对应 tag 构建多架构镜像并推送至 GHCR；随后在 `main` 上仅更新 `CHANGELOG.md`（将 `Unreleased` 提升为已发布版本，并确保新建空的 `Unreleased`），自动创建合并 PR

仓库设置建议：
- Workflow 权限：Settings → Actions → General → Workflow permissions 选择 “Read and write permissions”
- GHCR 推送权限：若仓库属于组织，默认 `GITHUB_TOKEN` 即可推送；否则可以配置 `GHCR_TOKEN`（PAT，包含 `write:packages`，并为组织开启 SSO）

## 许可证

MIT，详见 `LICENSE`。
