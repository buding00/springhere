package source

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/buding00/springhere/internal/clierr"
	"github.com/buding00/springhere/internal/safefs"
)

type Fetched struct {
	Kind    string
	URL     string
	Ref     string
	Commit  string
	Dir     string
	Cleanup func()
}

func FetchGit(ctx context.Context, kind, url, ref, wantCommit string) (*Fetched, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, clierr.Git("未找到 git，请先安装 Git 并保证在 PATH 中", err)
	}
	if url == "" || ref == "" {
		return nil, clierr.Git("git 源缺少 url 或 ref", nil)
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
	}

	bare, err := os.MkdirTemp("", "springhere-git-")
	if err != nil {
		return nil, err
	}
	cleanupBare := func() { _ = os.RemoveAll(bare) }

	if _, err := git(ctx, "-c", "init.defaultBranch=main", "init", "--bare", bare); err != nil {
		cleanupBare()
		return nil, clierr.Git("无法创建临时 git 仓库", err)
	}
	spec := ref + ":" + ref
	if _, err := git(ctx, "--git-dir="+bare, "fetch", "--depth", "1", "--no-tags", url, spec); err != nil {
		cleanupBare()
		return nil, clierr.Git(fetchHint(kind, url, ref, err), err)
	}
	got, err := git(ctx, "--git-dir="+bare, "rev-parse", ref)
	if err != nil {
		cleanupBare()
		return nil, clierr.Git("无法解析拉取到的 commit", err)
	}
	if wantCommit != "" && !strings.EqualFold(got, wantCommit) {
		cleanupBare()
		return nil, clierr.Git(fmt.Sprintf("commit 不匹配：注册表钉死 %s，%s 上 %s 实际是 %s。拒绝使用「最新 tag」凑合", wantCommit, kind, ref, got), nil)
	}

	extract, err := os.MkdirTemp("", "springhere-src-")
	if err != nil {
		cleanupBare()
		return nil, err
	}
	if err := archiveExtract(ctx, bare, ref, extract); err != nil {
		cleanupBare()
		_ = os.RemoveAll(extract)
		return nil, err
	}
	cleanupBare()

	return &Fetched{
		Kind:    kind,
		URL:     url,
		Ref:     ref,
		Commit:  got,
		Dir:     extract,
		Cleanup: func() { _ = os.RemoveAll(extract) },
	}, nil
}

func fetchHint(kind, url, ref string, err error) string {
	msg := fmt.Sprintf("从 %s 拉取 %s（%s）失败", kind, url, ref)
	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "not found") || strings.Contains(lower, "404"):
		if kind == "gitee" {
			return msg + "。Gitee 镜像可能尚未创建，可改用 --source github，或 --backend-template-dir / --frontend-template-dir 指向本地仓库"
		}
		return msg + "。请确认仓库公开可访问，或改用本地模板目录"
	case strings.Contains(lower, "timed out") || strings.Contains(lower, "could not resolve"):
		return msg + "。网络不可达时可换另一个源，或使用本地模板目录"
	default:
		return msg
	}
}

func archiveExtract(ctx context.Context, gitDir, ref, dest string) error {
	cmd := exec.CommandContext(ctx, "git", "--git-dir="+gitDir, "archive", "--format=tar", ref)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = gitEnv()
	if err := cmd.Run(); err != nil {
		return clierr.Git("git archive 失败: "+strings.TrimSpace(stderr.String()), err)
	}
	tr := tar.NewReader(bytes.NewReader(stdout.Bytes()))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return clierr.Git("读取 git archive 失败", err)
		}
		name := filepath.ToSlash(hdr.Name)
		if err := safefs.RelOK(name); err != nil {
			return clierr.Git("archive 含非法路径 "+name, err)
		}
		switch hdr.Typeflag {
		case tar.TypeXHeader, tar.TypeXGlobalHeader:
			continue
		case tar.TypeDir:
			if err := os.MkdirAll(filepath.Join(dest, filepath.FromSlash(name)), 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if hdr.FileInfo().Mode()&os.ModeSymlink != 0 {
				return clierr.Git("拒绝 archive 中的符号链接: "+name, nil)
			}
			path := filepath.Join(dest, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode().Perm())
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink, tar.TypeLink:
			return clierr.Git("拒绝 archive 中的链接: "+name, nil)
		default:
			return clierr.Git(fmt.Sprintf("拒绝 archive 中的文件类型 %v: %s", hdr.Typeflag, name), nil)
		}
	}
}

func git(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = gitEnv()
	if err := cmd.Run(); err != nil {
		out := strings.TrimSpace(stderr.String())
		if out == "" {
			out = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func gitEnv() []string {
	return append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=echo",
		"GCM_INTERACTIVE=never",
	)
}

func LocalCommit(dir string) string {
	out, err := git(context.Background(), "-C", dir, "rev-parse", "HEAD")
	if err != nil {
		return "local"
	}
	return out
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
