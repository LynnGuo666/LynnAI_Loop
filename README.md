# Loop

Anthropic API 密钥管理与代理网关。

管理多个 API 密钥，自动轮转负载均衡，探测密钥健康状态，记录用量统计。单二进制文件，内嵌前端，开箱即用。

## 功能

- **密钥轮转代理** — 轮转（Round-Robin）分配请求到多个 API 密钥，自动故障转移，支持流式和非流式请求
- **自动禁用与恢复** — 连续失败达到阈值自动禁用密钥，后台定期探测，恢复后自动重新启用
- **用量统计** — 记录每次请求的 token 用量、延迟、模型等信息，提供时序图表和按模型/渠道的统计分析
- **批量管理** — 批量导入/导出密钥，批量探测密钥状态
- **管理面板** — 内嵌 React SPA，中文界面，仪表盘、渠道管理、密钥管理、用量明细、统计分析、系统设置

## 快速开始

### Docker Compose（推荐）

```bash
mkdir loop && cd loop
curl -O https://raw.githubusercontent.com/LynnGuo666/LynnAI-Loop/main/docker-compose.yml
docker compose up -d
```

首次启动时，日志中会打印自动生成的管理员密钥：

```bash
docker compose logs -f loop
# Admin token: loop_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

访问 `http://localhost:8080`，使用该密钥登录。

### Docker

```bash
docker run -d \
  --name loop \
  -p 8080:8080 \
  -v loop-data:/data \
  -e DB_PATH=/data/loop.db \
  ghcr.io/lynnguo666/lynnai_loop:latest
```

### 本地编译

```bash
# 需要 Node.js 22+ 和 Go 1.23+
./build.sh
ADMIN_TOKEN=your-secret ./loop
```

## 配置

通过环境变量配置，所有参数均有默认值：

| 变量 | 默认值 | 说明 |
|---|---|---|
| `PORT` | `8080` | HTTP 监听端口 |
| `DB_PATH` | `loop.db` | SQLite 数据库路径 |
| `ADMIN_TOKEN` | 自动生成 | 管理员密钥，留空则首次启动时自动生成 |
| `DISABLE_THRESHOLD` | `5` | 连续失败多少次后自动禁用密钥 |
| `RECOVERY_PROBE_ENABLED` | `true` | 是否启用自动恢复探测 |
| `PROBE_BACKOFF_BASE_MIN` | `60` | 恢复探测初始退避时间（分钟） |
| `PROBE_BACKOFF_MAX_MIN` | `1440` | 恢复探测最大退避时间（分钟） |
| `PROBE_CHECK_INTERVAL_SEC` | `300` | 探测周期间隔（秒） |
| `MAX_PROXY_ATTEMPTS` | `5` | 单次请求最多尝试多少个密钥 |
| `UPSTREAM_TIMEOUT_SEC` | `300` | 非流式请求上游超时（秒） |
| `RESPONSE_HEADER_TIMEOUT_SEC` | `60` | 等待上游响应头超时（秒） |
| `MAX_REQUEST_BODY_MB` | `32` | 请求体大小上限（MB） |
| `DB_READ_POOL_SIZE` | `8` | SQLite 读连接池大小 |
| `PROXY_MAX_CONCURRENCY` | `200` | 最大并发代理请求数 |
| `PROXY_BACKLOG` | `100` | 超出并发限制后的排队上限 |
| `PROXY_BACKLOG_TIMEOUT_SEC` | `5` | 排队等待超时（秒） |
| `PROBE_BATCH_CONCURRENCY` | `10` | 批量探测并发数 |

## 使用方式

### 1. 创建渠道

渠道（Channel）代表一个 Anthropic API 端点，配置 `base_url`（如 `http://your-proxy` 或 `http://45.89.172.237:80`）。

### 2. 添加密钥

在渠道下添加一个或多个 API Key。系统会自动轮转使用这些密钥。

### 3. 发送请求

将客户端的 `base_url` 指向 Loop，使用任意一个有效的 API Key：

```bash
# 单渠道时可使用自动路由
curl http://localhost:8080/v1/messages \
  -H "x-api-key: your-api-key" \
  -H "content-type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{"model":"claude-sonnet-4-20250514","max_tokens":1024,"messages":[{"role":"user","content":"Hello"}]}'

# 多渠道时指定渠道 ID
curl http://localhost:8080/channel/1/v1/messages \
  -H "x-api-key: your-api-key" \
  -H "content-type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{"model":"claude-sonnet-4-20250514","max_tokens":1024,"messages":[{"role":"user","content":"Hello"}]}'
```

也可以使用 `Authorization: Bearer <token>` 头代替 `x-api-key`。

### 4. 查看用量

- **仪表盘** `/` — 总览渠道数、密钥数、请求量、近 7 天趋势
- **用量明细** `/usage` — 每次请求的详细日志，支持按渠道/模型/状态/时间筛选
- **统计分析** `/statistics` — 按模型和渠道的性能对比图表

## API 概览

所有接口需要 `Authorization: Bearer <token>` 或 `x-api-key` 认证。

| 接口 | 说明 |
|---|---|
| `POST /v1/messages` | 代理请求（单渠道自动路由） |
| `POST /channel/{id}/v1/messages` | 代理请求（指定渠道） |
| `GET /api/channels` | 渠道列表 |
| `POST /api/channels` | 创建渠道 |
| `GET /api/keys` | 密钥列表 |
| `POST /api/keys/import` | 批量导入密钥 |
| `POST /api/keys/probe` | 批量探测密钥 |
| `GET /api/usage` | 用量日志（分页） |
| `GET /api/usage/stats` | 用量统计 |
| `GET /api/usage/timeseries` | 用量时序数据 |
| `GET /api/settings` | 系统设置 |
| `PUT /api/settings` | 更新设置 |

## 技术栈

- **后端** — Go 1.23、Chi 路由、pure-Go SQLite（WAL 模式）
- **前端** — React 19、TypeScript、HeroUI v2、Tailwind CSS v4、Recharts
- **构建** — 前端 SPA 通过 `//go:embed` 内嵌到 Go 二进制，单文件部署
- **CI/CD** — GitHub Actions，自动构建 Docker 多架构镜像和跨平台二进制

## 本地开发

```bash
# 终端 1：启动后端
cd backend && go run .

# 终端 2：启动前端（Vite 开发服务器，端口 3000）
cd frontend && npm install && npm run dev
```

前端开发服务器会将 `/api`、`/channel`、`/v1` 请求代理到 `http://localhost:8080`。

