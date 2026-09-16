package source

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/buding00/springhere/internal/clierr"
	"github.com/buding00/springhere/internal/registry"
)

type Request struct {
	Kind          string
	LocalDir      string
	Component     *registry.Component
	PreferMirrors []string
	Detector      Detector
}

func Resolve(ctx context.Context, req Request) (*Fetched, string, error) {
	if req.LocalDir != "" {
		abs, err := filepath.Abs(req.LocalDir)
		if err != nil {
			return nil, "", clierr.Source("无法解析本地模板目录", err)
		}
		info, err := os.Stat(abs)
		if err != nil || !info.IsDir() {
			return nil, "", clierr.Source("本地模板目录不存在或不是目录: "+req.LocalDir, err)
		}
		return &Fetched{
			Kind:    "local",
			URL:     abs,
			Ref:     "local",
			Commit:  LocalCommit(abs),
			Dir:     abs,
			Cleanup: func() {},
		}, "使用本地模板 " + abs, nil
	}
	if req.Component == nil {
		return nil, "", clierr.Source("缺少组件", nil)
	}

	order := mirrorOrder(req.Kind, req.Detector)
	var last error
	var notes []string
	for i, kind := range order {
		src, err := req.Component.Source(kind)
		if err != nil {
			last = err
			continue
		}
		fetched, err := FetchGit(ctx, kind, src.URL, src.Ref, src.Commit)
		if err != nil {
			last = err
			if i+1 < len(order) {
				notes = append(notes, fmt.Sprintf("%s 失败，尝试 %s", kind, order[i+1]))
				continue
			}
			return nil, "", err
		}
		note := fmt.Sprintf("从 %s 拉取 %s @ %s", kind, src.URL, fetched.Commit)
		if len(notes) > 0 {
			note = notes[0] + "；" + note
		}
		return fetched, note, nil
	}
	if last == nil {
		last = clierr.Source("没有可用的模板源", nil)
	}
	return nil, "", last
}

func mirrorOrder(kind string, detect Detector) []string {
	picked, _ := PickMirror(kind, detect)
	if kind == "github" || kind == "gitee" {
		return []string{kind}
	}
	if picked == "gitee" {
		return []string{"gitee", "github"}
	}
	return []string{"github", "gitee"}
}
