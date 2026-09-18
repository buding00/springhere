package contract

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/buding00/springhere/internal/clierr"
)

const FileName = ".springhere-template.yaml"

type Contract struct {
	SchemaVersion int         `yaml:"schema_version"`
	Component     string      `yaml:"component"`
	CLIConstraint string      `yaml:"cli_constraint"`
	Payload       Payload     `yaml:"payload"`
	Transforms    []Transform `yaml:"transforms"`
	Raw           []byte      `yaml:"-"`
	SHA256        string      `yaml:"-"`
}

type Payload struct {
	Include []string `yaml:"include"`
}

type Transform struct {
	Kind      string `yaml:"kind"`
	Path      string `yaml:"path"`
	Key       string `yaml:"key"`
	From      string `yaml:"from"`
	ValueFrom string `yaml:"value_from"`
}

func Load(root string) (*Contract, error) {
	path := filepath.Join(root, FileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, clierr.Contract("缺少 "+FileName+"。从 GitHub/Gitee 拉取时请先把该文件推到对应仓库的 ref；本地开发请把 --backend-template-dir / --frontend-template-dir 指到组件仓库根目录", err)
	}
	c, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func Parse(data []byte) (*Contract, error) {
	var c Contract
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, clierr.Contract("无法解析 "+FileName, err)
	}
	c.Raw = append([]byte(nil), data...)
	sum := sha256.Sum256(data)
	c.SHA256 = fmt.Sprintf("sha256:%x", sum[:])
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Contract) validate() error {
	if c.SchemaVersion != 1 {
		return clierr.Contract(fmt.Sprintf("不支持的契约 schema_version %d", c.SchemaVersion), nil)
	}
	switch c.Component {
	case "backend", "frontend":
	default:
		return clierr.Contract("component 须是 backend 或 frontend", nil)
	}
	if len(c.Payload.Include) == 0 {
		return clierr.Contract("payload.include 不能为空", nil)
	}
	for _, p := range c.Payload.Include {
		if err := checkInclude(p); err != nil {
			return err
		}
	}
	for _, t := range c.Transforms {
		switch t.Kind {
		case "go_module", "json_string", "yaml_scalar", "deploy_stack":
		default:
			return clierr.Contract("未知 transform kind: "+t.Kind+"（支持 go_module、json_string、yaml_scalar、deploy_stack）", nil)
		}
		if t.Kind != "go_module" && strings.TrimSpace(t.Path) == "" {
			return clierr.Contract("transform "+t.Kind+" 缺少 path", nil)
		}
		if (t.Kind == "json_string" || t.Kind == "yaml_scalar") && t.Key == "" {
			return clierr.Contract("transform "+t.Kind+" 缺少 key", nil)
		}
		if t.Kind == "deploy_stack" {
			if err := checkDirName(t.From); err != nil {
				return clierr.Contract("transform deploy_stack 的 from 非法: "+t.From, err)
			}
		}
		if t.ValueFrom == "" {
			return clierr.Contract("transform "+t.Kind+" 缺少 value_from", nil)
		}
	}
	return nil
}

func checkInclude(p string) error {
	p = strings.TrimSpace(p)
	if p == "" || p == "." {
		return clierr.Contract("payload.include 不能为空或 .", nil)
	}
	if filepath.IsAbs(p) || strings.Contains(p, "..") {
		return clierr.Contract("payload.include 拒绝绝对路径和 ..: "+p, nil)
	}
	if strings.ContainsAny(p, `\:`) {
		return clierr.Contract("payload.include 含非法字符: "+p, nil)
	}
	return nil
}

func checkDirName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return fmt.Errorf("不能为空或 . / ..")
	}
	if strings.ContainsAny(name, `/\:`) {
		return fmt.Errorf("不能含路径分隔符")
	}
	return nil
}
