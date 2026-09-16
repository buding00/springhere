# SpringHere CLI 实施方案（最终）

> 状态：**当前执行基线**
>
> 范围：只设计并实现 `springhere` CLI。不包含 Axum 后端、Vue 前端的实现。
>
> 取代：[CLI实施方案.md](./CLI实施方案.md)（V1 执行稿）、[CLI实施方案V2-组件化架构.md](./CLI实施方案V2-组件化架构.md)（V2 愿景稿）。与仓库根目录 `docs/实施计划.md` 冲突时以本文为准。
>
> 架构依据：[架构文档版本4.md](../../docs/架构文档版本4.md) 的生成安全模型与渐进顺序；交互与多源吸收自 V2，但裁掉未经验证的组件市场。

## 1. 一句话目标

第一版 CLI 用类似 Vite 的交互（也可完全静默）把 **已经能跑的 Gin Server + React Admin** 原子地初始化成一个全栈项目。架构按「可替换组件」来搭，但注册表里暂时只放这一对；以后加 Axum / Vue 时扩注册表和组合编排，不重写生成核心。

## 2. 愿景与当前范围

### 2.1 愿景（本阶段不实现组件本身）

```text
用户执行 springhere new
  → 交互选择后端（Gin / 以后 Axum）
  → 交互选择前端（React / 以后 Vue）
  → 生成可运行的全栈项目
```

长期可以出现的组合：

| 后端 | 前端 | 何时进入 CLI |
|---|---|---|
| Gin (Go) | React Admin | **现在**，唯一已验收组合 |
| Axum (Rust) | React Admin | Axum 仓库独立完成，并与 React 源码联调通过之后 |
| Gin (Go) | Vue Admin | Vue 仓库独立完成，并与 Gin 源码联调通过之后 |
| Axum (Rust) | Vue Admin | 两个新组件都联调通过之后 |

Axum、Vue 的业务实现不在本仓库、不在本里程碑。本文只保证 CLI 的数据模型和扩展点不会把后路堵死。

### 2.2 当前必须交付

- 可执行命令 `springhere`
- 交互式 `springhere new`（Vite 风格）
- 非交互 `springhere new <name> --yes ...`（CI 友好）
- 只生成 **gin + react**
- GitHub / Gitee 作为同一 commit 的镜像
- 开发期可用本地模板目录，不依赖远程 tag
- 原子写入、dry-run、非空目录拒绝覆盖

### 2.3 当前明确不做

- 实现或内嵌 Axum / Vue / Spring Boot 源码
- 向导或 flag 里出现「即将推出」的空选项
- `--features`、RBAC、动态菜单、多租户
- `feature add` / `module add` / `upgrade`
- 远程组件市场、企业自定义源（不钉 commit 的任意 URL）
- 在 CLI 仓库另写一份 API 端点清单（`api-v1.yaml`）
- 初始化时执行 `go mod tidy`、`pnpm install`、migration
- `--backend-only` / `--frontend-only` 作为产品路径

## 3. 设计原则

1. **先产品、后模板、再组合。** CLI 只复制已被真实项目证明可用的组件，不在生成器里发明后端。
2. **组件可替换，组合须验收。** 后端、前端是独立 Git 仓库；能否配对由兼容性记录说了算，不能只凭「都声明了 v1」。
3. **CLI 不内嵌业务源码。** 二进制只带注册表、组合编排文件和生成逻辑。
4. **转换必须结构化。** Go 用 `modfile` + AST，JSON/YAML 用对应 parser。禁止全库字符串替换。源码中不出现破坏语法的 `{{.ProjectName}}`。
5. **一次写入，全部成功或全部没有。** 共用 staging，失败不留半项目。
6. **版本身份是 commit，URL 只是镜像。** Gitee 与 GitHub 必须指向同一完整 SHA。
7. **扩展点预留，假能力不预留。** 注册表可以加组件；向导不展示做不到的功能。

## 4. 核心概念

| 概念 | 含义 |
|---|---|
| Component | 一个可独立发布的后端或前端仓库，例如 `springhere-gin-server` |
| Template Contract | 组件根目录的 `.springhere-template.yaml`，声明白名单和封闭转换 |
| Combination | 一组已经联调过的 backend + frontend，以及根级 Compose/Makefile/Nginx |
| Registry | CLI 内嵌的组件与组合清单 |
| Source | 同一 commit 的获取地址：github / gitee / 本地目录 |
| Manifest | 生成项目根目录的 `.springhere.yaml`，记录用了什么，不宣称拥有业务源码 |

第一版只有一个 Combination：`gin-react-admin`。用户交互看到的是「选后端、选前端」，内部落到这个 combination。以后 Axum 进来时，是新增 combination，不是让用户任意两两拼接未测过的仓库。

## 5. 产品形态

### 5.1 交互式（默认，类似 Vite）

无参数或目标名可省略时进入问答。TTY 不可用且未传 `--yes` 时失败，避免 CI 挂起。

```text
$ springhere new

✔ 项目名称: › order-system
✔ 后端: › Gin (Go)
✔ 前端: › React + Ant Design
✔ Go module: › github.com/acme/order-system
✔ 模板源: › 自动检测

创建 order-system ...
下一步:
  cd order-system
  make help
```

第一版后端、前端都只有一项时仍然展示（和 Vite 在只有一种模板时的行为一致），这样第二项进来只需改注册表，不必改问答结构。

问答规则：

- 后端只有 `gin` 时，仍问 Go module（由所选后端的 `language: go` 决定，而不是写死「永远问 Go module」）
- 以后选 Axum 时，同一提示位改为 Cargo package 名（届时再加，第一版不必实现 Rust 转换）
- 不问 Feature、不问租户、不出现 Vue/Axum 占位项
- 源选项：自动检测 / GitHub / Gitee；若同时传了本地模板目录则跳过源选择

### 5.2 非交互式

```bash
springhere new order-system \
  --yes \
  --module github.com/acme/order-system
```

可选参数：

| 参数 | 默认 | 说明 |
|---|---|---|
| `--backend` | `gin` | 第一版只接受 `gin` |
| `--frontend` | `react` | 第一版只接受 `react` |
| `--module` | 必填（Go 后端时） | Go module 路径 |
| `--source` | `auto` | `auto` / `github` / `gitee` |
| `--backend-template-dir` | 空 | 本地后端模板，开发用 |
| `--frontend-template-dir` | 空 | 本地前端模板，开发用 |
| `--dry-run` | false | 只打印计划，不写目标 |
| `--no-git` | false | 不在目标根 `git init` |
| `--yes` | false | 跳过确认；非 TTY 必须带 |

未知 `--backend` / `--frontend` 立即失败，并提示当前已注册项。不要静默忽略。

### 5.3 其他命令（第一版）

```text
springhere                 # 帮助
springhere new             # 创建
springhere version         # 版本
```

`list` / `info` / `config` 第一版不做。需要看选项时用 `springhere new --help`。

### 5.4 生成结果

```text
order-system/
├── backend/                 # 来自 gin-server，module 已改写
├── frontend/                # 来自 react-admin，package name 已改写
├── deployments/             # 该 combination 的 Compose / Nginx / 前端镜像
├── .github/workflows/
├── .env.example
├── .springhere.yaml
├── Makefile
└── README.md
```

只有一个根 `.git`。`backend/`、`frontend/` 不保留模板仓库的 remote。根级 Compose 按真实 Gin 依赖包含 **PostgreSQL 和 Redis**。

## 6. 组件注册表

内嵌 `registry/components.yaml`。第一版只登记真实存在的仓库；Axum/Vue 的条目等组件就绪再加，不提前写空 URL。

示意（commit / URL 在发布前用占位，有 tag 再钉死）：

```yaml
schema_version: 1

backends:
  - id: gin
    name: "Gin (Go)"
    language: go
    sources:
      github:
        url: https://github.com/<owner>/springhere-gin-server.git
        ref: v0.2.0
        commit: "<full-sha>"
      gitee:
        url: https://gitee.com/<owner>/springhere-gin-server.git
        ref: v0.2.0
        commit: "<full-sha>"   # 必须与 github 相同
    requirements:
      go: ">=1.26"
      postgres: ">=14"
      redis: ">=7"

frontends:
  - id: react
    name: "React + Ant Design"
    language: typescript
    sources:
      github:
        url: https://github.com/<owner>/springhere-react-admin.git
        ref: v0.2.0
        commit: "<full-sha>"
      gitee:
        url: https://gitee.com/<owner>/springhere-react-admin.git
        ref: v0.2.0
        commit: "<full-sha>"
    requirements:
      node: ">=22"
      pnpm: ">=9"

combinations:
  - id: gin-react-admin
    backend: gin
    frontend: react
    tested: true          # 仅在该对已手工联调后为 true
    auth_mode: refresh-cookie
    api_base_path: /api
    # 权威契约是后端 OpenAPI，CLI 只钉摘要，不维护端点副本
    openapi_sha256: "sha256:..."
```

规则：

- `combinations[].tested` 为 false 的组合，交互与非交互都拒绝
- 未出现在 `combinations` 里的 backend×frontend 拒绝，即使两个组件各自存在
- 同一组件 github/gitee 的 `commit` 不一致则注册表非法，CLI 拒绝启动该组件
- 以后加 Axum：先有仓库和契约，再在 `backends` 加一条，再在 `combinations` 加 `axum-react-admin`（或实际 id），并补该组合的根级编排文件

不要在 CLI 内维护 `registry/contracts/api-v1.yaml` 这种端点清单。接口变化以组件仓库的 OpenAPI 为准。

## 7. 模板契约（组件仓库，CLI 消费）

两个现有仓库需要补（与 CLI 并行，否则只能用 testdata fixture）：

- `springhere-gin-server/.springhere-template.yaml`
- `springhere-react-admin/.springhere-template.yaml`

契约只声明 CLI 能安全执行的事：

```yaml
schema_version: 1
component: backend          # 或 frontend
cli_constraint: ">=0.1.0 <1.0.0"

payload:
  include:
    - cmd
    - internal
    - pkg
    - migrations
    - configs
    - deployments
    - .env.example
    - .gitignore
    - Makefile
    - go.mod
    - go.sum
    - README.md
    # 按真实仓库目录列白名单，不要抄 V4 示例树

transforms:
  - kind: go_module
    value_from: backend_module
  - kind: yaml_scalar
    path: configs/config.example.yaml   # 或实际配置文件路径
    key: app.name
    value_from: project_name
```

前端同理：`payload.include` 按现有 React 仓库；`transforms` 用 `json_string` 改 `package.json` 的 `name`。

约束：

- 白名单复制；排除 `.git`、`node_modules`、构建产物、本地 `.env`、secret
- 未知 `kind` 使初始化失败
- 第一版不做 `html_text`：`index.html` 保持通用标题
- 契约不能声明 shell、安装脚本或 hook
- `payload.include` 跟真实目录走（Gin 当前是 `pkg/`、`internal/data` 等），CLI 不写死文件清单

## 8. 多源

### 8.1 语义

GitHub 与 Gitee 是同一 commit 的两个取件处，不是两个版本。自动检测只选镜像，不选 tag。

优先级：

1. `--backend-template-dir` / `--frontend-template-dir`（开发）
2. 用户指定 `--source github|gitee`
3. `--source auto`：测连通性，选更低延迟的镜像；失败则尝试另一个
4. 取到对象后校验完整 commit；对不上则失败，不接受「该镜像最新 tag」

### 8.2 Git 操作

使用系统 `git`，参数走 argv 数组，禁止把用户输入拼进 shell。

推荐流程（与 V4 一致）：

```text
临时 bare clone（钉死 commit，而不是 depth-1 默认分支）
  → git archive
  → Go tar reader 解到 staging
  → 拒绝 symlink 与路径穿越
```

不要 `git clone --depth 1 <url>` 再 `checkout` 默认分支。镜像同步（GitHub → Gitee）是组件仓库的发布流程，不是 CLI 运行时逻辑；CLI 只校验 SHA。

### 8.3 第一版不做

- IP 归属地判断
- 用户级 `~/.springhere/config.yaml` 自定义源
- 持久化 cache / 离线模式

## 9. CLI 仓库目录

```text
springhere/
├── cmd/springhere/main.go
├── internal/
│   ├── command/          # root、new、版本、退出码
│   ├── prompt/           # 交互问答；非 TTY 检测
│   ├── registry/         # 加载内嵌 components.yaml，校验组合
│   ├── source/           # LocalDir / Git archive / 镜像选择
│   ├── contract/         # .springhere-template.yaml
│   ├── initializer/      # 复制、转换、根文件、manifest
│   │   ├── gomodule.go   # 第一版唯一语言转换
│   │   ├── structured.go # JSON / YAML
│   │   └── manifest.go
│   └── safefs/           # 路径边界、staging、原子 rename
├── registry/
│   └── components.yaml
├── combinations/
│   └── gin-react-admin/  # 只放根级编排，不放业务源码
│       ├── deployments/
│       ├── .github/workflows/
│       ├── .env.example
│       ├── Makefile
│       └── README.md
├── testdata/             # 缩小的假 backend/frontend
├── integration/
├── go.mod
└── README.md
```

包边界：

| 包 | 做 | 不做 |
|---|---|---|
| `command` | 参数、help、退出码 | 文件系统细节 |
| `prompt` | 问答；选项来自 registry | 写文件 |
| `registry` | 组件与 combination 合法性 | 拉取 Git |
| `source` | 本地目录或钉死 commit 的 archive | 拼 shell |
| `contract` | schema、白名单、未知 transform 拒绝 | hook |
| `initializer` | 转换 + 写根文件 + manifest | 跑组件仓库里的命令 |
| `safefs` | symlink 拒绝、staging、rename | 边渲染边写目标目录 |

以后加 Axum 时，新增的是 `initializer` 下的语言转换（例如 `cargo.go`）和 `combinations/axum-react-admin/`，而不是改 `safefs` 或重写 `new`。

## 10. 初始化流水线

```text
解析参数；无 TTY 且无 --yes 则失败
  → 交互或 flag 得到 backend、frontend、project、module、source
  → registry 查找 combination；未登记或 tested=false 则失败
  → 校验项目名、module、目标目录（非空则失败）
  → 解析两个源（本地 dir 或 git 镜像 + commit）
  → 读并校验两个 .springhere-template.yaml
  → 在临时 staging：
        按白名单导出 backend/
        按白名单导出 frontend/
        按契约做结构化转换
        写入该 combination 的根级文件
        写入 .springhere.yaml
  → 静态检查：无源 module/package 残留、无 .git、无越界路径
  → --dry-run 到此结束，打印将创建的文件树
  → 目标为空或不存在时原子 rename
  → 默认 git init 根仓库（--no-git 跳过）
```

任一环节失败：删除 staging，目标保持原样。对外文案是「创建失败，未写入目标」，不要先打印「后端已生成」。

## 11. Manifest

生成项目根 `.springhere.yaml`：

```yaml
schema_version: 1
project_name: order-system
cli_version: 0.1.0
created_at: "2026-09-15T00:00:00Z"

combination: gin-react-admin

components:
  backend:
    id: gin
    source: gitee          # 实际用的镜像；本地目录则为 local
    url: https://gitee.com/<owner>/springhere-gin-server.git
    ref: v0.2.0
    commit: "<full-sha>"
    contract_sha256: "sha256:..."
  frontend:
    id: react
    source: gitee
    url: https://gitee.com/<owner>/springhere-react-admin.git
    ref: v0.2.0
    commit: "<full-sha>"
    contract_sha256: "sha256:..."

backend:
  module: github.com/acme/order-system
  language: go

frontend:
  package_name: order-system-admin

auth_mode: refresh-cookie
api_base_path: /api
openapi_sha256: "sha256:..."

managed_files: {}
```

第一版不写 `features`、不写 `upgrade` 可升级列表。`managed_files` 为空：普通源码生成后归用户。

## 12. 以后加 Axum / Vue 时，CLI 要动什么

只记 CLI 侧清单，方便以后开工，本阶段不执行。

Axum 就绪后：

1. `registry/components.yaml` 增加 `backends` 条目（真实 URL + 同一 commit 的两个源）
2. 增加 `combinations` 条目，且仅在与目标前端联调后把 `tested` 设为 true
3. 新增 `combinations/axum-react-admin/`（或实际 id）根级编排：Cargo/compose/README 按 Axum 真实依赖写，不要复用 gin 的 Makefile 硬改
4. `initializer` 增加该语言的封闭转换（`Cargo.toml` 等），由模板契约的 `transforms.kind` 触发
5. 交互：registry 有两项后端时自然出现第二个选项；选中 `language: rust` 时问 package 名而不是 Go module
6. 未知语言、未知 transform 仍然失败，不回退到字符串替换

Vue 同理：加 `frontends`、加 combination、加根级编排；前端转换仍以 JSON/结构化文件为主。

在第二个真实组件出现之前，不要为它们预留空实现、空目录或 coming soon 选项。

## 13. 编码切片

均在 `springhere/`。不写业务模板。

### S0 骨架

`go.mod`、`cmd/springhere`、Cobra root、版本、统一错误与退出码（参数 / 未知组件 / 组合未验收 / 契约 / 源校验 / 写入冲突 / 外部 git）。质量门槛：`gofmt`、`go vet`、`go test`。

### S1 输入与交互

- flag 模型与默认值
- 交互问答（选项来自 registry）
- 非 TTY 无 `--yes` 失败
- 表驱动：非法名称、路径穿越、空 module、未知 backend/frontend、未登记组合

### S2 注册表与安全复制

- 加载 `components.yaml`
- testdata 假组件：白名单、拒绝 symlink、拒绝 `../`、未知 transform 失败、dry-run 不写盘

### S3 结构化转换

- Go module + 内部 import
- YAML 标量、JSON `package.json` name
- 生成结果无源 module / 源 package name

### S4 本地全量

对着相邻 `springhere-gin-server`、`springhere-react-admin` 跑 `new`（需两个仓库已有契约文件）。验收文件树、manifest、go.mod、package.json、根 Compose（含 Redis）、无 `.git` 残留。生成目录内分别跑后端测试和前端 build，CLI 进程不代跑安装。

### S5 Git 镜像

bare clone + archive；`--source github|gitee|auto`；commit 不一致拒绝；干净环境可复现。无远程时用 git fixture。

### S6 组合冒烟

生成项目 Compose 拉起 Postgres/Redis + 后端 + Nginx 前端；login → refresh → me → logout。同一 CLI 多次生成结果一致。

依赖：S0–S3 可立即开始；S4 依赖组件契约；S5–S6 依赖可引用 commit（先本地/fixture，有 tag 再钉进 registry）。

## 14. 质量与安全

沿用 V1/V4 的工程约定：

- `new` 不覆盖非空目录
- `--dry-run` 不写目标
- 禁止执行模板仓库提供的脚本
- 错误信息可操作，不打印 token、Cookie、DSN
- 合并前：`gofmt -w`、`go vet ./...`、`go test ./...`、`git diff --check`

退出码可区分：参数错误、未知或不允许的组合、契约非法、源/commit 不匹配、目标冲突、git 失败。

## 15. 风险

| 风险 | 控制 |
|---|---|
| 组件目录继续变化 | 白名单写在组件契约里，不写死在 CLI |
| 尚无远程 tag | 本地 `--*-template-dir` 为开发主路径 |
| 自动源选错镜像 | SHA 校验；CI 显式 `--source` |
| 过早抽象多语言 | 第一版只实现 `go_module`；其他 kind 失败 |
| 把愿景选项做进向导 | 注册表没有的组件不展示 |
| 根编排绑死 Gin | 编排按 combination 分目录，不按「通用一份 tmpl」 |

## 16. 编码前冻结项

1. CLI `go.mod` 路径（例如 `github.com/<owner>/springhere`）
2. 组件 Git URL 的 owner；没有远程则第一版只支持本地模板目录，registry 里 URL 可占位
3. 两个 `.springhere-template.yaml` 的 `payload.include` 按**现有**仓库目录起草
4. 交互库：`survey/v2` 或 `charmbracelet/huh`，实现时二选一
5. 根 Compose 含 PostgreSQL **和** Redis（按真实 Gin，不是旧文档的仅 Postgres）

确认后第一批代码：S0 + S1，做到 `springhere --help`、`springhere new --help`、交互骨架和输入校验测试。
