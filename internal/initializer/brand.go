package initializer

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/buding00/springhere/internal/clierr"
	"github.com/buding00/springhere/internal/contract"
)

var markAssignRe = regexp.MustCompile(`(mark:\s*)(?:'[^']*'|"[^"]*")`)

func rewriteAppBrand(root string, t contract.Transform, project string) error {
	project = strings.TrimSpace(project)
	from := strings.TrimSpace(t.From)
	if project == "" {
		return clierr.Contract("app_brand 需要项目名称", nil)
	}
	cfgPath := filepath.Join(root, filepath.FromSlash(t.Path))
	if err := replaceBrandToken(cfgPath, from, project); err != nil {
		return err
	}
	if err := rewriteBrandMark(cfgPath, project); err != nil {
		return err
	}
	for _, name := range []string{"index.html", "AGENTS.md"} {
		extra := filepath.Join(root, name)
		if _, err := os.Stat(extra); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if err := replaceBrandToken(extra, from, project); err != nil {
			return err
		}
	}
	return nil
}

func replaceBrandToken(path, from, to string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return clierr.Contract("无法读取 "+filepath.ToSlash(path), err)
	}
	if !strings.Contains(string(data), from) {
		return clierr.Contract("app_brand 在 "+filepath.Base(path)+" 中未找到 "+from, nil)
	}
	out := strings.ReplaceAll(string(data), from, to)
	return os.WriteFile(path, []byte(out), 0o644)
}

func rewriteBrandMark(path, project string) error {
	mark, err := brandMark(project)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return clierr.Contract("无法读取 "+filepath.ToSlash(path), err)
	}
	loc := markAssignRe.FindSubmatchIndex(data)
	if loc == nil {
		return clierr.Contract("app_brand 未找到 mark 字段", nil)
	}
	prefix := data[loc[2]:loc[3]]
	full := data[loc[0]:loc[1]]
	if len(full) <= len(prefix) {
		return clierr.Contract("app_brand mark 字段格式无效", nil)
	}
	quote := full[len(prefix)]
	repl := append([]byte{}, prefix...)
	repl = append(repl, quote)
	repl = append(repl, mark...)
	repl = append(repl, quote)
	out := append([]byte{}, data[:loc[0]]...)
	out = append(out, repl...)
	out = append(out, data[loc[1]:]...)
	return os.WriteFile(path, out, 0o644)
}

func brandMark(project string) (string, error) {
	for _, r := range project {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return string(unicode.ToUpper(r)), nil
		}
	}
	return "", clierr.Contract("app_brand 无法从项目名得到 mark", nil)
}
