package initializer

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/buding00/springhere/internal/clierr"
	"github.com/buding00/springhere/internal/contract"
	"github.com/buding00/springhere/internal/safefs"
)

var composeFileNames = []string{
	"docker-compose.yaml",
	"docker-compose.yml",
	"compose.yaml",
	"compose.yml",
}

func rewriteDeployStack(root string, t contract.Transform, project string) error {
	project = strings.TrimSpace(project)
	if project == "" {
		return clierr.Contract("deploy_stack 需要项目名称", nil)
	}
	from := strings.TrimSpace(t.From)
	base := filepath.Join(root, filepath.FromSlash(t.Path))
	src := filepath.Join(base, from)
	dst := filepath.Join(base, project)

	info, err := os.Stat(src)
	if err != nil || !info.IsDir() {
		return clierr.Contract("deploy_stack 找不到目录 "+filepath.ToSlash(filepath.Join(t.Path, from)), err)
	}
	if from != project {
		if _, err := os.Stat(dst); err == nil {
			return clierr.Contract("deploy_stack 目标已存在 "+filepath.ToSlash(filepath.Join(t.Path, project)), nil)
		}
		if err := os.Rename(src, dst); err != nil {
			return clierr.Contract("无法将 "+from+" 重命名为 "+project, err)
		}
	}

	changed := 0
	for _, name := range composeFileNames {
		path := filepath.Join(dst, name)
		n, err := rewriteComposeFile(path, from, project)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		changed += n
	}
	if from != project && changed == 0 {
		return clierr.Contract("deploy_stack 未在 compose 中找到容器名前缀 "+from+"-", nil)
	}
	return nil
}

func rewriteComposeFile(path, from, to string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return 0, clierr.Contract("无法解析 "+filepath.Base(path), err)
	}
	n := rewriteContainerNames(&doc, from, to)
	if n == 0 {
		return 0, nil
	}
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		_ = enc.Close()
		return 0, err
	}
	if err := enc.Close(); err != nil {
		return 0, err
	}
	if err := os.WriteFile(path, []byte(buf.String()), 0o644); err != nil {
		return 0, err
	}
	return n, nil
}

func rewriteContainerNames(n *yaml.Node, from, to string) int {
	if n == nil {
		return 0
	}
	switch n.Kind {
	case yaml.DocumentNode, yaml.SequenceNode:
		c := 0
		for _, ch := range n.Content {
			c += rewriteContainerNames(ch, from, to)
		}
		return c
	case yaml.MappingNode:
		c := 0
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, v := n.Content[i], n.Content[i+1]
			if k.Value == "container_name" && v.Kind == yaml.ScalarNode {
				if next, ok := retargetContainerName(v.Value, from, to); ok && next != v.Value {
					v.Value = next
					c++
				}
				continue
			}
			c += rewriteContainerNames(v, from, to)
		}
		return c
	default:
		return 0
	}
}

func retargetContainerName(name, from, to string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == from {
		return to, true
	}
	prefix := from + "-"
	if strings.HasPrefix(name, prefix) {
		return to + "-" + strings.TrimPrefix(name, prefix), true
	}
	return name, false
}

func listRelFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if safefs.IsExcluded(rel, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
