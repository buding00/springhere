package validate

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/mod/module"

	"github.com/buding00/springhere/internal/clierr"
)

var projectNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

func ProjectName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return clierr.Usage("项目名称不能为空")
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return clierr.Usage("项目名称不能包含路径分隔符")
	}
	if !projectNameRe.MatchString(name) {
		return clierr.Usage("项目名称须为小写字母开头，只含小写字母、数字和连字符，最长 63 个字符")
	}
	if strings.Contains(name, "--") || strings.HasSuffix(name, "-") {
		return clierr.Usage("项目名称不能以连字符结尾，也不能包含连续连字符")
	}
	return nil
}

func GoModule(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return clierr.Usage("Go module 路径不能为空（例如 github.com/acme/order-system）")
	}
	if err := module.CheckPath(path); err != nil {
		return clierr.Usagef("非法的 Go module 路径 %q: %v", path, err)
	}
	return nil
}

func SourceKind(kind string) error {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "auto", "github", "gitee":
		return nil
	default:
		return clierr.Usage("来源须是 auto、github 或 gitee")
	}
}

func OutputDir(dir string) error {
	if dir == "" {
		return clierr.Usage("输出目录不能为空")
	}
	clean := filepath.Clean(dir)
	if clean == "." || clean == string(filepath.Separator) {
		return clierr.Usage("不能把当前目录或根目录当作新项目输出路径")
	}
	for _, r := range dir {
		if r == 0 || unicode.IsControl(r) {
			return clierr.Usage("输出目录含非法字符")
		}
	}
	return nil
}

func NPMPackageName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, "_", "-")
	if name == "" {
		return "app-admin"
	}
	if !strings.HasSuffix(name, "-admin") {
		name += "-admin"
	}
	return name
}

func JoinHint(ids []string) string {
	if len(ids) == 0 {
		return "(无)"
	}
	quoted := make([]string, 0, len(ids))
	for _, id := range ids {
		quoted = append(quoted, fmt.Sprintf("%q", id))
	}
	return strings.Join(quoted, ", ")
}
