package initializer

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/buding00/springhere/internal/clierr"
)

func rewriteGoModule(root, newModule string) (oldModule string, err error) {
	modPath := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(modPath)
	if err != nil {
		return "", clierr.Contract("go_module 转换需要 go.mod", err)
	}
	parsed, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return "", clierr.Contract("无法解析 go.mod", err)
	}
	if parsed.Module == nil {
		return "", clierr.Contract("go.mod 缺少 module 语句", nil)
	}
	oldModule = parsed.Module.Mod.Path
	if oldModule == newModule {
		return oldModule, nil
	}
	if err := parsed.AddModuleStmt(newModule); err != nil {
		return "", err
	}
	formatted, err := parsed.Format()
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(modPath, formatted, 0o644); err != nil {
		return "", err
	}
	if err := rewriteGoImports(root, oldModule, newModule); err != nil {
		return "", err
	}
	if err := patchFile(filepath.Join(root, "AGENTS.md"), oldModule, newModule); err != nil {
		return "", err
	}
	return oldModule, nil
}

func rewriteGoImports(root, oldModule, newModule string) error {
	fset := token.NewFileSet()
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == "vendor" || base == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		if err != nil {
			rel, _ := filepath.Rel(root, path)
			return clierr.Contract("无法解析 Go 文件 "+rel, err)
		}
		changed := false
		for _, imp := range file.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				continue
			}
			np, ok := rewriteImportPath(p, oldModule, newModule)
			if !ok {
				continue
			}
			imp.Path.Value = strconv.Quote(np)
			changed = true
		}
		if !changed {
			return nil
		}
		var buf bytes.Buffer
		if err := format.Node(&buf, fset, file); err != nil {
			return err
		}
		return os.WriteFile(path, buf.Bytes(), 0o644)
	})
}

func rewriteImportPath(p, oldModule, newModule string) (string, bool) {
	if p == oldModule {
		return newModule, true
	}
	if strings.HasPrefix(p, oldModule+"/") {
		return newModule + strings.TrimPrefix(p, oldModule), true
	}
	return p, false
}

func leftoverGoModule(root, oldModule string) error {
	if oldModule == "" {
		return nil
	}
	var hits []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		name := d.Name()
		if name != "go.mod" && name != "AGENTS.md" && !strings.HasSuffix(name, ".go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if !bytes.Contains(data, []byte(oldModule)) {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		hits = append(hits, filepath.ToSlash(rel))
		return nil
	})
	if len(hits) > 0 {
		return clierr.Contract(fmt.Sprintf("生成结果仍含源 module %q，出现在: %s", oldModule, strings.Join(hits, ", ")), nil)
	}
	return nil
}
