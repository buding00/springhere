package command

import (
	"context"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/buding00/springhere/internal/clierr"
	"github.com/buding00/springhere/internal/initializer"
	"github.com/buding00/springhere/internal/prompt"
	"github.com/buding00/springhere/internal/registry"
	"github.com/buding00/springhere/internal/validate"
)

func newNewCommand(stdout io.Writer) *cobra.Command {
	var (
		backend     string
		frontend    string
		module      string
		sourceKind  string
		backendDir  string
		frontendDir string
		dryRun      bool
		noGit       bool
		yes         bool
		output      string
	)

	cmd := &cobra.Command{
		Use:   "new [project-name]",
		Short: "创建 Gin + React 全栈项目",
		Long: `创建已验收组合 gin-react-admin。

交互模式（类似 Vite）：
  springhere new

非交互（CI / 脚本必须加 --yes）：
  springhere new order-system --yes --module github.com/acme/order-system

模板源：
  --source auto     按连通性在 GitHub / Gitee 间选择，失败则尝试另一个
  --source github   只从 GitHub 拉取
  --source gitee    只从 Gitee 拉取（镜像须与 GitHub 为同一 commit）

本地开发不必拉取远程：
  --backend-template-dir ../springhere-gin-server \
  --frontend-template-dir ../springhere-react-admin
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 1 {
				name = args[0]
			}
			reg, err := registry.Load()
			if err != nil {
				return err
			}

			localBoth := backendDir != "" && frontendDir != ""
			needPrompt := !yes
			if needPrompt && !prompt.Interactive() {
				return clierr.Usage("非交互环境请添加 --yes，并提供项目名称和 --module（Go 后端）")
			}
			ans := prompt.Answers{
				ProjectName: name,
				Backend:     backend,
				Frontend:    frontend,
				Module:      module,
				Source:      sourceKind,
			}
			if needPrompt {
				ans, err = prompt.Ask(reg, ans, localBoth)
				if err != nil {
					return err
				}
			} else {
				if ans.Backend == "" {
					ans.Backend = "gin"
				}
				if ans.Frontend == "" {
					ans.Frontend = "react"
				}
				if ans.Source == "" {
					ans.Source = "auto"
				}
				if ans.ProjectName == "" {
					return clierr.Usage("请提供项目名称，例如 springhere new order-system --yes --module github.com/acme/order-system")
				}
			}

			if err := validate.ProjectName(ans.ProjectName); err != nil {
				return err
			}
			backendComp := reg.Backend(ans.Backend)
			if backendComp == nil {
				return clierr.Component("未知后端 " + ans.Backend + "，可选：" + validate.JoinHint(reg.BackendIDs()))
			}
			if backendComp.Language == "go" {
				if err := validate.GoModule(ans.Module); err != nil {
					return err
				}
			}
			if (backendDir == "") != (frontendDir == "") {
				return clierr.Usage("--backend-template-dir 与 --frontend-template-dir 需要同时提供，或同时省略以使用 GitHub/Gitee")
			}

			target := output
			if target == "" {
				target = ans.ProjectName
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
			defer cancel()

			_, err = initializer.Run(ctx, initializer.Config{
				ProjectName:         ans.ProjectName,
				TargetDir:           target,
				Backend:             ans.Backend,
				Frontend:            ans.Frontend,
				Module:              ans.Module,
				Source:              ans.Source,
				BackendTemplateDir:  backendDir,
				FrontendTemplateDir: frontendDir,
				DryRun:              dryRun,
				NoGit:               noGit,
				Stdout:              stdout,
			})
			return err
		},
	}

	cmd.Flags().StringVar(&backend, "backend", "gin", "后端组件 id（第一版仅 gin）")
	cmd.Flags().StringVar(&frontend, "frontend", "react", "前端组件 id（第一版仅 react）")
	cmd.Flags().StringVar(&module, "module", "", "Go module 路径")
	cmd.Flags().StringVar(&sourceKind, "source", "auto", "模板源：auto、github、gitee")
	cmd.Flags().StringVar(&backendDir, "backend-template-dir", "", "本地后端模板目录")
	cmd.Flags().StringVar(&frontendDir, "frontend-template-dir", "", "本地前端模板目录")
	cmd.Flags().StringVar(&output, "output", "", "输出目录（默认 ./<project-name>）")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "只打印将创建的文件，不写入目标")
	cmd.Flags().BoolVar(&noGit, "no-git", false, "不在 <项目名>_backend / <项目名>_frontend 执行 git init")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "跳过交互确认（非 TTY 必须）")
	return cmd
}
