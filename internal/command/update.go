package command

import (
	"context"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/buding00/springhere/internal/prompt"
	"github.com/buding00/springhere/internal/selfupdate"
)

func newUpdateCommand(stdout io.Writer) *cobra.Command {
	var (
		yes   bool
		check bool
		ver   string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "从 GitHub Release 更新当前 CLI",
		Long: `从 GitHub Release 下载当前系统对应的二进制，替换正在使用的 springhere。

只使用 GitHub，不访问 Gitee，也不会在本机拉取源码或编译。

  springhere update
  springhere update --check
  springhere update --version v0.1.0
  springhere update --yes

非终端环境若需要真正下载安装，必须加 --yes。第一次安装请到
https://github.com/buding00/springhere/releases 下载对应平台的包。
`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 3*time.Minute)
			defer cancel()
			return selfupdate.Run(ctx, selfupdate.Options{
				Check:       check,
				Yes:         yes,
				Interactive: prompt.Interactive(),
				Version:     ver,
				Stdout:      stdout,
				Confirm:     prompt.Confirm,
			})
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "跳过确认（非 TTY 下载时必须）")
	cmd.Flags().BoolVar(&check, "check", false, "只比较版本，不下载")
	cmd.Flags().StringVar(&ver, "version", "", "安装指定 GitHub Release tag，例如 v0.1.0")
	return cmd
}
