package registry

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/buding00/springhere/internal/clierr"
	"github.com/buding00/springhere/internal/embedded"
	"github.com/buding00/springhere/internal/validate"
)

type Registry struct {
	SchemaVersion int           `yaml:"schema_version"`
	Backends      []Component   `yaml:"backends"`
	Frontends     []Component   `yaml:"frontends"`
	Combinations  []Combination `yaml:"combinations"`
}

type Component struct {
	ID           string            `yaml:"id"`
	Name         string            `yaml:"name"`
	Language     string            `yaml:"language"`
	Sources      map[string]Source `yaml:"sources"`
	Requirements map[string]string `yaml:"requirements"`
}

type Source struct {
	URL    string `yaml:"url"`
	Ref    string `yaml:"ref"`
	Commit string `yaml:"commit"`
}

type Combination struct {
	ID            string `yaml:"id"`
	Backend       string `yaml:"backend"`
	Frontend      string `yaml:"frontend"`
	Tested        bool   `yaml:"tested"`
	AuthMode      string `yaml:"auth_mode"`
	APIBasePath   string `yaml:"api_base_path"`
	OpenAPISha256 string `yaml:"openapi_sha256"`
}

func Load() (*Registry, error) {
	return Parse(embedded.ComponentsYAML)
}

func Parse(data []byte) (*Registry, error) {
	var r Registry
	if err := yaml.Unmarshal(data, &r); err != nil {
		return nil, clierr.Contract("无法解析组件注册表", err)
	}
	if err := r.validate(); err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *Registry) validate() error {
	if r.SchemaVersion != 1 {
		return clierr.Contract(fmt.Sprintf("不支持的注册表 schema_version %d", r.SchemaVersion), nil)
	}
	if err := validateComponents("backend", r.Backends); err != nil {
		return err
	}
	if err := validateComponents("frontend", r.Frontends); err != nil {
		return err
	}
	seenCombo := map[string]struct{}{}
	for _, c := range r.Combinations {
		if c.ID == "" || c.Backend == "" || c.Frontend == "" {
			return clierr.Contract("combination 缺少 id/backend/frontend", nil)
		}
		if _, ok := seenCombo[c.ID]; ok {
			return clierr.Contract("重复的 combination id: "+c.ID, nil)
		}
		seenCombo[c.ID] = struct{}{}
		if r.Backend(c.Backend) == nil {
			return clierr.Contract("combination "+c.ID+" 引用了未知后端 "+c.Backend, nil)
		}
		if r.Frontend(c.Frontend) == nil {
			return clierr.Contract("combination "+c.ID+" 引用了未知前端 "+c.Frontend, nil)
		}
	}
	return nil
}

func validateComponents(kind string, list []Component) error {
	seen := map[string]struct{}{}
	for _, c := range list {
		if c.ID == "" {
			return clierr.Contract(kind+" 缺少 id", nil)
		}
		if _, ok := seen[c.ID]; ok {
			return clierr.Contract("重复的 "+kind+" id: "+c.ID, nil)
		}
		seen[c.ID] = struct{}{}
		gh, hasGH := c.Sources["github"]
		ge, hasGE := c.Sources["gitee"]
		if !hasGH && !hasGE {
			return clierr.Contract(c.ID+" 至少需要 github 或 gitee 源", nil)
		}
		if hasGH {
			if err := checkSource(c.ID, "github", gh); err != nil {
				return err
			}
		}
		if hasGE {
			if err := checkSource(c.ID, "gitee", ge); err != nil {
				return err
			}
		}
		if hasGH && hasGE {
			gc := strings.TrimSpace(gh.Commit)
			ec := strings.TrimSpace(ge.Commit)
			if gc != "" && ec != "" && gc != ec {
				return clierr.Contract(c.ID+" 的 github/gitee commit 必须相同（镜像，不是两个版本）", nil)
			}
			if (gc == "") != (ec == "") {
				return clierr.Contract(c.ID+" 的 github 与 gitee 必须同时钉死 commit，或同时留空使用 ref tip", nil)
			}
		}
	}
	return nil
}

func checkSource(id, name string, s Source) error {
	if strings.TrimSpace(s.URL) == "" || strings.TrimSpace(s.Ref) == "" {
		return clierr.Contract(id+" 的 "+name+" 源缺少 url 或 ref", nil)
	}
	return nil
}

func (r *Registry) Backend(id string) *Component {
	return find(r.Backends, id)
}

func (r *Registry) Frontend(id string) *Component {
	return find(r.Frontends, id)
}

func find(list []Component, id string) *Component {
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	return nil
}

func (r *Registry) BackendIDs() []string {
	return ids(r.Backends)
}

func (r *Registry) FrontendIDs() []string {
	return ids(r.Frontends)
}

func ids(list []Component) []string {
	out := make([]string, 0, len(list))
	for _, c := range list {
		out = append(out, c.ID)
	}
	return out
}

func (r *Registry) Combination(backend, frontend string) (*Combination, error) {
	if r.Backend(backend) == nil {
		return nil, clierr.Component(fmt.Sprintf("未知后端 %q，可选：%s", backend, validate.JoinHint(r.BackendIDs())))
	}
	if r.Frontend(frontend) == nil {
		return nil, clierr.Component(fmt.Sprintf("未知前端 %q，可选：%s", frontend, validate.JoinHint(r.FrontendIDs())))
	}
	for i := range r.Combinations {
		c := &r.Combinations[i]
		if c.Backend == backend && c.Frontend == frontend {
			if !c.Tested {
				return nil, clierr.Combine(fmt.Sprintf("组合 %s+%s 尚未验收（tested=false），拒绝生成", backend, frontend))
			}
			return c, nil
		}
	}
	return nil, clierr.Combine(fmt.Sprintf("没有已验收的组合 %s + %s", backend, frontend))
}

func (c *Component) Source(kind string) (Source, error) {
	s, ok := c.Sources[kind]
	if !ok {
		keys := make([]string, 0, len(c.Sources))
		for k := range c.Sources {
			keys = append(keys, k)
		}
		return Source{}, clierr.Source(fmt.Sprintf("组件 %s 没有源 %s，可选：%s", c.ID, kind, strings.Join(keys, ", ")), nil)
	}
	return s, nil
}
