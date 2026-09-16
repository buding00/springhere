package command

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/buding00/springhere/internal/clierr"
	"github.com/buding00/springhere/internal/version"
)

func Execute() error {
	return NewRoot(os.Stdin, os.Stdout, os.Stderr).Execute()
}

func NewRoot(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "springhere",
		Short:         "SpringHere 全栈项目脚手架",
		Long:          "生成已验收的 Gin + React 全栈项目。模板从 GitHub 或 Gitee 镜像拉取，也可使用本地组件仓库。",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Version,
	}
	cmd.SetIn(stdin)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetVersionTemplate("springhere {{.Version}}\n")
	cmd.AddCommand(newNewCommand(stdout))
	cmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "打印版本",
		Run: func(c *cobra.Command, args []string) {
			_, _ = fmt.Fprintf(stdout, "springhere %s\n", version.Version)
		},
	})
	return cmd
}

func Exit(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(clierr.ExitCode(err))
}
