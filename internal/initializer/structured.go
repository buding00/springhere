package initializer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/buding00/springhere/internal/clierr"
	"github.com/buding00/springhere/internal/contract"
)

type values struct {
	ProjectName         string
	BackendModule       string
	FrontendPackageName string
}

func (v values) lookup(name string) (string, error) {
	switch name {
	case "project_name":
		return v.ProjectName, nil
	case "backend_module":
		return v.BackendModule, nil
	case "frontend_package_name":
		return v.FrontendPackageName, nil
	default:
		return "", clierr.Contract("未知 value_from: "+name, nil)
	}
}

func applyTransforms(root string, transforms []contract.Transform, v values) error {
	for _, t := range transforms {
		val, err := v.lookup(t.ValueFrom)
		if err != nil {
			return err
		}
		switch t.Kind {
		case "go_module":
			if _, err := rewriteGoModule(root, val); err != nil {
				return err
			}
		case "json_string":
			if err := setJSONString(filepath.Join(root, t.Path), t.Key, val); err != nil {
				return err
			}
		case "yaml_scalar":
			if err := setYAMLScalar(filepath.Join(root, t.Path), t.Key, val); err != nil {
				return err
			}
		case "deploy_stack":
			if err := rewriteDeployStack(root, t, val); err != nil {
				return err
			}
		case "app_brand":
			if err := rewriteAppBrand(root, t, val); err != nil {
				return err
			}
		default:
			return clierr.Contract("未知 transform kind: "+t.Kind, nil)
		}
	}
	return nil
}

func setJSONString(path, key, value string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return clierr.Contract("无法读取 "+path, err)
	}
	out, err := replaceTopLevelJSONString(data, key, value)
	if err != nil {
		return clierr.Contract("无法更新 JSON "+path, err)
	}
	return os.WriteFile(path, out, 0o644)
}

func replaceTopLevelJSONString(data []byte, key, value string) ([]byte, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	if _, ok := obj[key]; !ok {
		return nil, fmt.Errorf("缺少顶层字段 %q", key)
	}
	quoted, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	needle := []byte(`"` + key + `"`)
	idx := bytes.Index(data, needle)
	if idx < 0 {
		return nil, fmt.Errorf("未找到字段 %q", key)
	}
	colon := bytes.IndexByte(data[idx+len(needle):], ':')
	if colon < 0 {
		return nil, fmt.Errorf("字段 %q 格式错误", key)
	}
	start := idx + len(needle) + colon + 1
	for start < len(data) && (data[start] == ' ' || data[start] == '\t' || data[start] == '\n' || data[start] == '\r') {
		start++
	}
	if start >= len(data) || data[start] != '"' {
		return nil, fmt.Errorf("字段 %q 不是字符串", key)
	}
	end := start + 1
	escaped := false
	for end < len(data) {
		c := data[end]
		if escaped {
			escaped = false
			end++
			continue
		}
		if c == '\\' {
			escaped = true
			end++
			continue
		}
		if c == '"' {
			end++
			break
		}
		end++
	}
	var buf bytes.Buffer
	buf.Write(data[:start])
	buf.Write(quoted)
	buf.Write(data[end:])
	if !json.Valid(buf.Bytes()) {
		return nil, fmt.Errorf("替换后 JSON 非法")
	}
	return buf.Bytes(), nil
}

func setYAMLScalar(path, dottedKey, value string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return clierr.Contract("无法读取 "+path, err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return clierr.Contract("无法解析 YAML "+path, err)
	}
	keys := strings.Split(dottedKey, ".")
	if err := setYAMLNode(&doc, keys, value); err != nil {
		return clierr.Contract("无法设置 YAML "+dottedKey, err)
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return err
	}
	_ = enc.Close()
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func setYAMLNode(n *yaml.Node, keys []string, value string) error {
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		return setYAMLNode(n.Content[0], keys, value)
	}
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("不是 mapping")
	}
	want := keys[0]
	for i := 0; i+1 < len(n.Content); i += 2 {
		k := n.Content[i]
		v := n.Content[i+1]
		if k.Value != want {
			continue
		}
		if len(keys) == 1 {
			v.Kind = yaml.ScalarNode
			v.Tag = "!!str"
			v.Value = value
			return nil
		}
		return setYAMLNode(v, keys[1:], value)
	}
	return fmt.Errorf("缺少键 %s", want)
}
