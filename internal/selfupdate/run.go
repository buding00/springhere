package selfupdate

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/buding00/springhere/internal/clierr"
	"github.com/buding00/springhere/internal/version"
)

type Options struct {
	Check          bool
	Yes            bool
	Interactive    bool
	Version        string
	CurrentVersion string
	GOOS           string
	GOARCH         string
	Executable     string
	Stdout         io.Writer
	Client         *Client
	Confirm        func(message string, def bool) (bool, error)
}

func Run(ctx context.Context, opts Options) error {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Client == nil {
		opts.Client = NewClient(nil)
	}
	if opts.GOOS == "" {
		opts.GOOS = runtime.GOOS
	}
	if opts.GOARCH == "" {
		opts.GOARCH = runtime.GOARCH
	}
	if opts.CurrentVersion == "" {
		opts.CurrentVersion = version.Display()
	}
	if opts.Executable == "" {
		exe, err := os.Executable()
		if err != nil {
			return clierr.Source("无法确定当前 CLI 路径", err)
		}
		opts.Executable = exe
	}

	releases, err := opts.Client.ListReleases(ctx)
	if err != nil {
		return err
	}

	var target *Release
	if strings.TrimSpace(opts.Version) != "" {
		target, err = FindRelease(releases, opts.Version)
	} else {
		target, err = LatestStable(releases)
	}
	if err != nil {
		return err
	}

	currentTag := NormalizeTag(opts.CurrentVersion)
	targetTag := NormalizeTag(target.Tag)
	exe := displayPath(opts.Executable)

	fmt.Fprintf(opts.Stdout, "当前 CLI: %s\n", displayCurrent(currentTag, opts.CurrentVersion))
	fmt.Fprintf(opts.Stdout, "目标版本: %s", targetTag)
	if target.HTMLURL != "" {
		fmt.Fprintf(opts.Stdout, "（%s）", target.HTMLURL)
	}
	fmt.Fprintln(opts.Stdout)

	if opts.Check {
		switch {
		case currentTag == "":
			fmt.Fprintln(opts.Stdout, "当前不是按 Release 安装的版本，可更新")
		case currentTag == targetTag:
			fmt.Fprintf(opts.Stdout, "已是最新稳定版 springhere %s\n", strings.TrimPrefix(targetTag, "v"))
		case semver.IsValid(currentTag) && semver.Compare(currentTag, targetTag) > 0:
			fmt.Fprintf(opts.Stdout, "当前版本新于 GitHub 稳定版 %s\n", targetTag)
		default:
			fmt.Fprintln(opts.Stdout, "可更新")
		}
		return nil
	}

	if currentTag != "" && currentTag == targetTag {
		fmt.Fprintf(opts.Stdout, "已是最新稳定版 springhere %s\n", strings.TrimPrefix(targetTag, "v"))
		return nil
	}

	downgrade := currentTag != "" && semver.IsValid(currentTag) && semver.Compare(currentTag, targetTag) > 0
	fmt.Fprintf(opts.Stdout, "将替换: %s\n", exe)
	if downgrade {
		fmt.Fprintf(opts.Stdout, "这是回退（当前 %s → %s）\n", currentTag, targetTag)
	}

	if !opts.Yes {
		if !opts.Interactive {
			return clierr.Usage("非交互环境请添加 --yes")
		}
		if opts.Confirm == nil {
			return clierr.Usage("已取消")
		}
		msg := fmt.Sprintf("确认安装 %s？", targetTag)
		def := !downgrade
		ok, err := opts.Confirm(msg, def)
		if err != nil {
			return err
		}
		if !ok {
			return clierr.Usage("已取消")
		}
	}

	asset, err := target.FindAsset(opts.GOOS, opts.GOARCH)
	if err != nil {
		return err
	}
	sums, err := target.FindChecksums()
	if err != nil {
		return err
	}

	sumBody, err := opts.Client.Download(ctx, sums.URL)
	if err != nil {
		return err
	}
	wantSum, err := parseChecksum(sumBody, asset.Name)
	if err != nil {
		return err
	}
	pkg, err := opts.Client.Download(ctx, asset.URL)
	if err != nil {
		return err
	}
	if err := verifySHA256(pkg, wantSum); err != nil {
		return err
	}
	binary, err := extractBinary(pkg, opts.GOOS, asset.Name)
	if err != nil {
		return err
	}
	if err := replaceExecutable(opts.Executable, binary); err != nil {
		return err
	}
	fmt.Fprintf(opts.Stdout, "已安装 springhere %s\n", strings.TrimPrefix(targetTag, "v"))
	return nil
}

func displayCurrent(tag, raw string) string {
	if tag != "" {
		return strings.TrimPrefix(tag, "v")
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "未知"
	}
	return raw + "（非 Release 安装）"
}

func displayPath(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		if resolved, err := filepath.EvalSymlinks(abs); err == nil {
			return resolved
		}
		return abs
	}
	return p
}
