# __PROJECT_NAME__

由 SpringHere CLI 生成的全栈项目：Gin 后端 + React 管理端。

## 目录

```text
backend/      Gin 服务
frontend/     React 管理端
deployments/  PostgreSQL、Redis、Nginx
```

## 前置条件

- Go 1.26+
- Node.js 22+ 与 pnpm
- Docker（用于 PostgreSQL 和 Redis）
- 系统已安装 `git`

## 本地开发

```bash
# 1. 启动 PostgreSQL 和 Redis，并整理 backend 依赖（go mod tidy）
make up

# 2. 后端：迁移并启动
make migrate
make backend

# 3. 另开终端启动前端（Vite 把 /api 代理到 localhost:8080）
make frontend
```

前端默认 http://localhost:5173。使用后端仓库里已有的管理账户登录。

模板来自 GitHub 或 Gitee 镜像（同一 commit）。本项目生成后不再依赖 SpringHere CLI。

## 常用命令

```bash
make help
make up          # 启动 Postgres + Redis，并 go mod tidy
make down        # 停止依赖
make migrate     # 执行数据库迁移
make backend     # 启动 Gin
make frontend    # 启动 React
```

后端 module：`__BACKEND_MODULE__`  
前端 package：`__FRONTEND_PACKAGE__`

## 配置

- 后端配置：`backend/application.yaml`，也可用环境变量覆盖（见 `backend/.env.example`）
- 前端代理：`frontend/.env.example` 中的 `VITE_API_PROXY_TARGET`

不要把密钥提交进 Git。生产环境必须替换 JWT Secret，并打开 Refresh Cookie 的 `Secure`。
