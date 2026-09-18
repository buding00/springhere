# __PROJECT_NAME__

由 SpringHere CLI 生成的全栈项目：Gin 后端 + React 管理端。

## 目录

```text
__BACKEND_DIR__/      Gin 服务，PostgreSQL / Redis / Nginx 编排在 __BACKEND_DIR__/deploy/__PROJECT_NAME__/
__FRONTEND_DIR__/     React 管理端
```

`__BACKEND_DIR__/` 与 `__FRONTEND_DIR__/` 各自是独立的 Git 仓库。根目录没有 `.git`。

## 前置条件

- Go 1.26+
- Node.js 22+ 与 pnpm
- Docker（用于 PostgreSQL 和 Redis）
- 系统已安装 `git`

## 本地开发

```bash
# 1. 启动 PostgreSQL 和 Redis
docker compose -f __BACKEND_DIR__/deploy/__PROJECT_NAME__/docker-compose.yaml up -d

# 2. 后端：整理依赖、迁移并启动
cd __BACKEND_DIR__
go mod tidy
go run ./cmd/migrate -action up
go run ./cmd/server -config application.yaml

# 3. 另开终端启动前端（Vite 把 /api 代理到 localhost:8080）
cd __FRONTEND_DIR__
pnpm install
pnpm dev
```

前端默认 http://localhost:5173。使用后端仓库里已有的管理账户登录。

模板来自 GitHub 或 Gitee 镜像（同一提交）。本项目生成后不再依赖 SpringHere CLI。

## 配置

- 后端配置：`__BACKEND_DIR__/application.yaml`，也可用环境变量覆盖（见 `__BACKEND_DIR__/.env.example`）
- 前端代理：`__FRONTEND_DIR__/.env.example` 中的 `VITE_API_PROXY_TARGET`
- Compose / Nginx：`__BACKEND_DIR__/deploy/__PROJECT_NAME__/`

不要把密钥提交进 Git。生产环境必须替换 JWT Secret，并打开 Refresh Cookie 的 `Secure`。

后端 module：`__BACKEND_MODULE__`  
前端 package：`__FRONTEND_PACKAGE__`
