package selfupdate

import (
	"fmt"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/buding00/springhere/internal/clierr"
)

func NormalizeTag(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	if v == "" || v == "devel" || v == "(devel)" {
		return ""
	}
	return "v" + v
}

func IsStableTag(tag string) bool {
	tag = NormalizeTag(tag)
	return semver.IsValid(tag) && semver.Prerelease(tag) == ""
}

func AssetName(tag, goos, goarch string) string {
	tag = NormalizeTag(tag)
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("springhere_%s_%s_%s%s", tag, goos, goarch, ext)
}

func ChecksumFileName() string {
	return "checksums.txt"
}

func LatestStable(releases []Release) (*Release, error) {
	var best *Release
	bestTag := ""
	for i := range releases {
		r := &releases[i]
		if r.Draft || r.Pre {
			continue
		}
		tag := NormalizeTag(r.Tag)
		if !IsStableTag(tag) {
			continue
		}
		if bestTag == "" || semver.Compare(tag, bestTag) > 0 {
			best = r
			bestTag = tag
		}
	}
	if best == nil {
		return nil, clierr.Source("GitHub 上没有稳定 Release（vX.Y.Z）。请先给 CLI 仓库打 tag 并等待 Release 工作流完成", nil)
	}
	best.Tag = bestTag
	return best, nil
}

func FindRelease(releases []Release, want string) (*Release, error) {
	want = NormalizeTag(want)
	if want == "" {
		return nil, clierr.Usage("--version 须是 vX.Y.Z，例如 --version v0.1.0")
	}
	for i := range releases {
		r := &releases[i]
		if r.Draft {
			continue
		}
		if NormalizeTag(r.Tag) == want {
			r.Tag = want
			return r, nil
		}
	}
	return nil, clierr.Source("没有 GitHub Release "+want+"。请确认 tag 已推送且 Release 已创建", nil)
}

func (r Release) FindAsset(goos, goarch string) (Asset, error) {
	want := AssetName(r.Tag, goos, goarch)
	var names []string
	for _, a := range r.Assets {
		names = append(names, a.Name)
		if a.Name == want {
			return a, nil
		}
	}
	have := strings.Join(names, ", ")
	if have == "" {
		have = "（无资产）"
	}
	return Asset{}, clierr.Source(fmt.Sprintf("Release %s 没有当前平台的安装包（%s/%s，需要 %s）。已有：%s", r.Tag, goos, goarch, want, have), nil)
}

func (r Release) FindChecksums() (Asset, error) {
	for _, a := range r.Assets {
		if a.Name == ChecksumFileName() {
			return a, nil
		}
	}
	return Asset{}, clierr.Source("Release "+r.Tag+" 缺少 checksums.txt，拒绝安装", nil)
}
