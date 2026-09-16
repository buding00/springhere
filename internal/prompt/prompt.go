package prompt

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"golang.org/x/term"

	"github.com/buding00/springhere/internal/clierr"
	"github.com/buding00/springhere/internal/registry"
	"github.com/buding00/springhere/internal/validate"
)

type Answers struct {
	ProjectName string
	Backend     string
	Frontend    string
	Module      string
	Source      string
}

func Interactive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func Ask(reg *registry.Registry, in Answers, skipSource bool) (Answers, error) {
	out := in
	if out.Backend == "" {
		out.Backend = "gin"
	}
	if out.Frontend == "" {
		out.Frontend = "react"
	}
	if out.Source == "" {
		out.Source = "auto"
	}

	qs := []*survey.Question{}
	if out.ProjectName == "" {
		qs = append(qs, &survey.Question{
			Name: "project",
			Prompt: &survey.Input{
				Message: "项目名称:",
				Help:    "小写字母开头，可含数字和连字符",
			},
			Validate: func(ans any) error {
				s, _ := ans.(string)
				return validate.ProjectName(s)
			},
		})
	}
	qs = append(qs, &survey.Question{
		Name: "backend",
		Prompt: &survey.Select{
			Message: "后端:",
			Options: componentLabels(reg.Backends),
			Default: labelFor(reg.Backends, out.Backend),
		},
	})
	qs = append(qs, &survey.Question{
		Name: "frontend",
		Prompt: &survey.Select{
			Message: "前端:",
			Options: componentLabels(reg.Frontends),
			Default: labelFor(reg.Frontends, out.Frontend),
		},
	})

	raw := struct {
		Project  string
		Backend  string
		Frontend string
		Module   string
		Source   string
	}{}
	if err := survey.Ask(qs, &raw); err != nil {
		return Answers{}, clierr.Usage("已取消")
	}
	if out.ProjectName == "" {
		out.ProjectName = raw.Project
	}
	if id := idFor(reg.Backends, raw.Backend); id != "" {
		out.Backend = id
	}
	if id := idFor(reg.Frontends, raw.Frontend); id != "" {
		out.Frontend = id
	}

	backend := reg.Backend(out.Backend)
	if backend != nil && backend.Language == "go" && out.Module == "" {
		def := "github.com/example/" + out.ProjectName
		modQ := []*survey.Question{{
			Name: "module",
			Prompt: &survey.Input{
				Message: "Go module 路径:",
				Default: def,
			},
			Validate: func(ans any) error {
				s, _ := ans.(string)
				return validate.GoModule(s)
			},
		}}
		if err := survey.Ask(modQ, &raw); err != nil {
			return Answers{}, clierr.Usage("已取消")
		}
		out.Module = raw.Module
	}

	if !skipSource {
		srcQ := []*survey.Question{{
			Name: "source",
			Prompt: &survey.Select{
				Message: "模板源:",
				Options: []string{
					"自动检测（推荐）",
					"GitHub (github.com)",
					"Gitee (gitee.com)",
				},
				Default: sourceLabel(out.Source),
			},
		}}
		if err := survey.Ask(srcQ, &raw); err != nil {
			return Answers{}, clierr.Usage("已取消")
		}
		out.Source = parseSourceLabel(raw.Source)
	}
	return out, nil
}

func componentLabels(list []registry.Component) []string {
	out := make([]string, 0, len(list))
	for _, c := range list {
		out = append(out, fmt.Sprintf("%s (%s)", c.Name, c.ID))
	}
	return out
}

func labelFor(list []registry.Component, id string) string {
	for _, c := range list {
		if c.ID == id {
			return fmt.Sprintf("%s (%s)", c.Name, c.ID)
		}
	}
	if len(list) > 0 {
		return fmt.Sprintf("%s (%s)", list[0].Name, list[0].ID)
	}
	return ""
}

func idFor(list []registry.Component, label string) string {
	for _, c := range list {
		if fmt.Sprintf("%s (%s)", c.Name, c.ID) == label || c.ID == label {
			return c.ID
		}
	}
	return ""
}

func sourceLabel(kind string) string {
	switch kind {
	case "github":
		return "GitHub (github.com)"
	case "gitee":
		return "Gitee (gitee.com)"
	default:
		return "自动检测（推荐）"
	}
}

func parseSourceLabel(label string) string {
	switch label {
	case "GitHub (github.com)":
		return "github"
	case "Gitee (gitee.com)":
		return "gitee"
	default:
		return "auto"
	}
}
