# SpringHere

一条命令生成可运行的全栈项目：Gin 后端 + React 管理端。

创建过程类似 Vite，可以问答，也可以把参数写全交给脚本。当前只提供这一套已对接好的组合。

## 需要什么

- [Go](https://go.dev/dl/) 1.26 或更高
- [Git](https://git-scm.com/)
- 生成之后跑项目还需要：Docker、Node.js 22+、[pnpm](https://pnpm.io/)

## 安装

到 [GitHub Releases](https://github.com/buding00/springhere/releases) 下载当前系统的包，解压后把 `springhere` 放到 `PATH` 里。资产名形如 `springhere_v0.1.0_darwin_arm64.tar.gz`。

已经装过之后，用 CLI 自己更新（只从 GitHub Release 下载，不在本机编译，也不走 Gitee）：

```bash
springhere update --check
springhere update
springhere update --yes
springhere update --version v0.1.0
```

非终端里真正下载安装必须加 `--yes`。已是最新版时不加也可以。

开发者也可以从源码安装：

```bash
git clone https://github.com/buding00/springhere.git
cd springhere
go install ./cmd/springhere
```

确认：

```bash
springhere version
springhere new --help
```

如果 `springhere` 找不到，把可执行文件所在目录加进 `PATH`（`go install` 一般是 `$(go env GOPATH)/bin`）。也可以不安装，在本仓库里执行 `go run ./cmd/springhere`。

## 创建项目

在终端里直接问：

```bash
springhere new
```

会询问项目名称、后端、前端、Go module 路径，以及从 GitHub 还是 Gitee 拉取模板。

参数一次写完（适合脚本）：

```bash
springhere new my-app --yes --module github.com/acme/my-app
```

`--yes` 跳过问答。不在终端里跑时必须加，否则会直接退出。

### 模板从哪来

默认 `--source auto`：测一下 GitHub 和 Gitee 哪个更快，失败会换另一个。国内网络不稳定时可以写死：

```bash
springhere new my-app --yes --module github.com/acme/my-app --source gitee
springhere new my-app --yes --module github.com/acme/my-app --source github
```

Gitee 是 GitHub 的备份镜像，内容应与 GitHub 同一提交。只想看将生成哪些文件、先不落地：

```bash
springhere new my-app --yes --module github.com/acme/my-app --dry-run
```

目标目录已有文件时不会覆盖。中途失败不会留下半成品。

## 生成结果

```text
my-app/
├── backend/              Gin 服务
├── frontend/             React 管理端
├── deployments/          PostgreSQL、Redis、Nginx
├── Makefile
├── README.md
└── .springhere.yaml      记录这次用了哪份模板
```

默认会在项目根执行 `git init`（只要一个 Git 仓库）。不想初始化就加 `--no-git`。

SpringHere 生成时不会替你装依赖。进入项目后，`make up` 会启动数据库并执行 `go mod tidy`，`make frontend` 会执行 `pnpm install`。

## 接下来怎么跑

```bash
cd my-app
make up          # 启动 PostgreSQL 和 Redis，并整理 backend 依赖
make migrate     # 数据库迁移
make backend     # 另开一个终端，Gin 默认 :8080
make frontend    # 再开一个终端，React 默认 :5173，/api 代理到后端
```

浏览器打开前端地址，用后端文档里的管理员账号登录。更完整的说明在生成项目的 `README.md`。

## 常用选项

| 选项 | 默认 | 说明 |
|---|---|---|
| `--yes` / `-y` | 关 | 跳过问答 |
| `--module` | 无 | Go module 路径，后端必填 |
| `--source` | `auto` | `auto`、`github` 或 `gitee` |
| `--dry-run` | 关 | 只打印文件列表，不写磁盘 |
| `--no-git` | 关 | 不在项目根执行 `git init` |
| `--output` | `./<项目名>` | 输出目录 |

第一版后端只能是 `gin`，前端只能是 `react`（也是默认值）。

---

## 从源码开发 CLI

给改 SpringHere 本身的人。普通创建项目用不到下面这些。

```bash
go test ./...
go vet ./...
go build -o bin/springhere ./cmd/springhere
```

用旁边尚未发布的组件仓库当模板：

```bash
springhere new demo --yes --module github.com/acme/demo \
  --backend-template-dir ../springhere-gin-server \
  --frontend-template-dir ../springhere-react-admin
```

发布一版给 `springhere update` 用：把 `internal/version/version.go` 改成目标版本（例如 `0.1.0`），打 **不可变** tag `v0.1.0` 并推到 GitHub。`.github/workflows/release.yml` 会交叉编译并挂到该 tag 的 GitHub Release。tag 已在但 Release 丢了：Actions 里手动跑 Release 工作流，填同一个 tag。不要 `git tag -f` 改已经发出去的 tag。

设计说明见 [docs/CLI实施方案最终.md](docs/CLI实施方案最终.md)。
