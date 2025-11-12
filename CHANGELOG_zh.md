# 更新日志

本项目遵循语义化版本（Semantic Versioning），日志格式参考 Keep a Changelog。

## [未发布]

- 暂无变更。

## [0.2.0] - 2025-11-12


### 新增
- WS：在全部 WebSocket 服务中增加外卖业务模拟端点
  - `ws://.../ws/food/user` 与 `ws://.../ws/food/merchant`
  - 不同服务采用不同事件键名：`type`（ws-echo）、`action`（ws-ticker）、`event`（ws-timeline）
  - 支持 `?interval=` 控制推送频率；支持静态资源驱动并在缺失时回退到内置数据
- HTTP：新增 RESTful 示例接口（三台 HTTP 服务均可用）
  - `GET|POST|OPTIONS /rest/items`
  - `GET|PUT|PATCH|DELETE|OPTIONS /rest/items/{id}`，使用进程内存存储
- 数据：新增静态 JSON 资源目录，可通过 `ASSETS_DIR` 覆盖
  - 端点可从 `assets/` 读取（如 `user/info.json`、`user/posts.json`、`order/orders.json`、`payment/checkout.json`、`ws/timeline.json`、`ws/food_*.json`、`rest/items.json`）
- 文档：完善中文接口清单 `docs/接口清单.md`
  - 列出完整端口与路径、WS 令牌用法、REST 端点、静态资源来源、WS 事件键名规则

### 变更
- WS 安全：所有 WS 端点统一启用静态令牌校验
  - 接受请求头 `X-Auth-Token: zhongmiao-org-token` 或查询参数 `?token=zhongmiao-org-token`
- HTTP/WS 行为：在存在静态资源时优先返回资源数据，否则回退到内置默认值

### 修复
- 文档：将 `docs/接口清单.md` 中的 JSON 示例改为合法 JSON（数组/对象），移除不规范占位符


## [0.1.1] - 2025-11-06

### 变更
- CI：releaseDraft 现在检查推送到 `main` 的所有提交是否修改了 `VERSION`/`CHANGELOG.md`（不再仅限 `head_commit`）
- CI：合并 `release.yml` 为单一作业，从发布 tag 构建镜像并随后提交 PR 更新 `CHANGELOG.md`
- CI：发布后流程不再自动修改 `VERSION`，仅更新 `CHANGELOG.md`
- 文档：统一 README 头部（居中样式、Go 与项目徽标），放大图标并对齐中英文徽章排版

### 新增
- 文档：新增 `README_zh.md`，并在 `README.md` 增加语言切换
- 文档：补充发布流程与 GHCR 发布说明
- 资源：新增按主题切换的加号图标 `docs/plus-light.svg` 与 `docs/plus-dark.svg`，用于 README 头部
- 徽标：README 切换为 SVG 项目标识（`docs/logo.svg`）以提升清晰度

## [0.1.0] - 2025-11-06

### 新增
- Go 1.25 模块：用于启动 Intercept Wave 的轻量上游测试编排器
- 3 个 HTTP 服务（端口：`BASE_PORT..BASE_PORT+2`，默认 9000–9002），提供通用调试接口：
  - `GET /` – 服务信息
  - `GET /health`
  - `GET /status/{code}`
  - `GET /delay/{ms}`
  - `POST|PUT|PATCH /echo` – 回显方法、路径、查询与请求体
  - `GET /headers` – 回显部分请求头
  - `GET /cookies` – 回显请求 Cookie
  - `GET /large?size=` – 返回大 JSON 载荷（上限 2MB）
- 服务特定的模拟接口（与 Intercept Wave 示例对齐）：
  - user-service（端口 `BASE`）：`GET /api/user/info`、`GET /api/posts`
  - order-service（端口 `BASE+1`）：`GET /order-api/orders`、`POST /order-api/orders`（创建并返回 id）、`GET /order-api/order/{id}/submit`
  - payment-service（端口 `BASE+2`）：`POST /pay-api/checkout`（模拟延迟）
- 3 个 WebSocket 服务（端口 `BASE+3..BASE+5`，默认 9003–9005）：
  - `ws://.../ws/echo` – 回显
  - `ws://.../ws/ticker?interval=1000` – 周期推送 `tick N`
  - `ws://.../ws/timeline` – 依次发送 `hello`、`processing`、`done` 并正常关闭
- 环境变量 `BASE_PORT`：同时控制 HTTP 与 WS 端口段
- Docker 多阶段构建：输出最小静态镜像（`scratch`），并提供 `.dockerignore`
- `docker-compose.yml`：暴露 9000–9005 端口，支持配置 `BASE_PORT`
- CI 工作流（`ci.yml`）：build、vet、test、gofmt 检查；推送到 `main` 从 `CHANGELOG.md` 的 Unreleased 生成 Release Draft
- 发布工作流（`release.yml`）：多架构（amd64/arm64）构建并推送到 GHCR `ghcr.io/zhongmiao-org/intercept-wave-upstream`
- 发布后自动化：把 `Unreleased` 提升到发布段落，确保 `CHANGELOG.md` 追加新的 `Unreleased`，并自动创建可合并 PR
- 文档：初始化 README（容器用法、端点示例、与 Intercept Wave 搭配说明）；初始化 CHANGELOG
- 单元测试：
  - HTTP：验证服务启动及 `GET /order-api/orders` 返回 `{ code: 0, ... }`
  - WS：验证 echo 连接与消息回显

