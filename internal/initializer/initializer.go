package initializer

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/buding00/springhere/internal/clierr"
	"github.com/buding00/springhere/internal/contract"
	"github.com/buding00/springhere/internal/embedded"
	"github.com/buding00/springhere/internal/registry"
	"github.com/buding00/springhere/internal/safefs"
	"github.com/buding00/springhere/internal/source"
	"github.com/buding00/springhere/internal/validate"
)

type Config struct {
	ProjectName         string
	TargetDir           string
	Backend             string
	Frontend            string
	Module              string
	FrontendPackage     string
	Source              string
	BackendTemplateDir  string
	FrontendTemplateDir string
	DryRun              bool
	NoGit               bool
	Stdout              io.Writer
	Detector            source.Detector
}

type Result struct {
	Files  []string
	Notes  []string
	Target string
	DryRun bool
}

func Run(ctx context.Context, cfg Config) (*Result, error) {
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	if err := validate.ProjectName(cfg.ProjectName); err != nil {
		return nil, err
	}
	if cfg.FrontendPackage == "" {
		cfg.FrontendPackage = validate.NPMPackageName(cfg.ProjectName)
	}
	if err := validate.GoModule(cfg.Module); err != nil {
		return nil, err
	}
	if err := validate.SourceKind(cfg.Source); err != nil {
		return nil, err
	}
	if cfg.TargetDir == "" {
		cfg.TargetDir = cfg.ProjectName
	}
	absTarget, err := filepath.Abs(cfg.TargetDir)
	if err != nil {
		return nil, err
	}
	cfg.TargetDir = absTarget
	if err := validate.OutputDir(cfg.TargetDir); err != nil {
		return nil, err
	}

	reg, err := registry.Load()
	if err != nil {
		return nil, err
	}
	combo, err := reg.Combination(cfg.Backend, cfg.Frontend)
	if err != nil {
		return nil, err
	}
	backendComp := reg.Backend(cfg.Backend)
	frontendComp := reg.Frontend(cfg.Frontend)

	if !cfg.DryRun {
		if err := safefs.TargetWritable(cfg.TargetDir); err != nil {
			return nil, err
		}
	}

	backendSrc, bNote, err := source.Resolve(ctx, source.Request{
		Kind:      cfg.Source,
		LocalDir:  cfg.BackendTemplateDir,
		Component: backendComp,
		Detector:  cfg.Detector,
	})
	if err != nil {
		return nil, err
	}
	defer backendSrc.Cleanup()

	frontendSrc, fNote, err := source.Resolve(ctx, source.Request{
		Kind:      cfg.Source,
		LocalDir:  cfg.FrontendTemplateDir,
		Component: frontendComp,
		Detector:  cfg.Detector,
	})
	if err != nil {
		return nil, err
	}
	defer frontendSrc.Cleanup()

	bContract, err := contract.Load(backendSrc.Dir)
	if err != nil {
		return nil, err
	}
	if bContract.Component != "backend" {
		return nil, clierr.Contract("后端模板契约 component 应为 backend", nil)
	}
	fContract, err := contract.Load(frontendSrc.Dir)
	if err != nil {
		return nil, err
	}
	if fContract.Component != "frontend" {
		return nil, clierr.Contract("前端模板契约 component 应为 frontend", nil)
	}

	parent := filepath.Dir(cfg.TargetDir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return nil, err
	}
	staging, err := os.MkdirTemp(parent, ".springhere-staging-*")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(staging) }()

	backendRoot := filepath.Join(staging, "backend")
	frontendRoot := filepath.Join(staging, "frontend")
	if err := os.MkdirAll(backendRoot, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(frontendRoot, 0o755); err != nil {
		return nil, err
	}

	bFiles, err := safefs.CopyAllowlist(backendSrc.Dir, backendRoot, bContract.Payload.Include)
	if err != nil {
		return nil, err
	}
	fFiles, err := safefs.CopyAllowlist(frontendSrc.Dir, frontendRoot, fContract.Payload.Include)
	if err != nil {
		return nil, err
	}

	sourceModule, _ := readGoModule(backendRoot)
	vals := values{
		ProjectName:         cfg.ProjectName,
		BackendModule:       cfg.Module,
		FrontendPackageName: cfg.FrontendPackage,
	}
	if err := applyTransforms(backendRoot, bContract.Transforms, vals); err != nil {
		return nil, err
	}
	if err := applyTransforms(frontendRoot, fContract.Transforms, vals); err != nil {
		return nil, err
	}
	bFiles, err = listRelFiles(backendRoot)
	if err != nil {
		return nil, err
	}
	fFiles, err = listRelFiles(frontendRoot)
	if err != nil {
		return nil, err
	}
	if sourceModule != "" && sourceModule != cfg.Module {
		if err := leftoverGoModule(backendRoot, sourceModule); err != nil {
			return nil, err
		}
	}

	rootFiles, err := writeCombination(staging, combo.ID, map[string]string{
		"__PROJECT_NAME__":     cfg.ProjectName,
		"__BACKEND_MODULE__":   cfg.Module,
		"__FRONTEND_PACKAGE__": cfg.FrontendPackage,
	})
	if err != nil {
		return nil, err
	}

	if err := writeManifest(staging, newManifest(cfg, combo, backendSrc, frontendSrc, bContract.SHA256, fContract.SHA256)); err != nil {
		return nil, err
	}

	var all []string
	for _, f := range bFiles {
		all = append(all, "backend/"+f)
	}
	for _, f := range fFiles {
		all = append(all, "frontend/"+f)
	}
	all = append(all, rootFiles...)
	all = append(all, ".springhere.yaml")
	sort.Strings(all)

	notes := []string{bNote, fNote}
	res := &Result{Files: all, Notes: notes, Target: cfg.TargetDir, DryRun: cfg.DryRun}

	fmt.Fprintf(cfg.Stdout, "组合: %s（%s + %s）\n", combo.ID, cfg.Backend, cfg.Frontend)
	for _, n := range notes {
		if strings.TrimSpace(n) != "" {
			fmt.Fprintln(cfg.Stdout, n)
		}
	}
	if cfg.DryRun {
		fmt.Fprintf(cfg.Stdout, "dry-run：不会写入 %s。将创建 %d 个文件：\n", cfg.TargetDir, len(all))
		for _, f := range all {
			fmt.Fprintf(cfg.Stdout, "  %s\n", f)
		}
		return res, nil
	}

	if err := safefs.Commit(staging, cfg.TargetDir); err != nil {
		return nil, err
	}

	if !cfg.NoGit {
		for _, name := range []string{"backend", "frontend"} {
			dir := filepath.Join(cfg.TargetDir, name)
			if err := gitInit(dir); err != nil {
				fmt.Fprintf(cfg.Stdout, "警告：%s git init 失败（项目已生成）: %v\n", name, err)
			}
		}
	}

	fmt.Fprintf(cfg.Stdout, "已创建 %s（%d 个文件）\n", cfg.TargetDir, len(all))
	fmt.Fprintf(cfg.Stdout, "\n下一步：\n  cd %s\n  查看 README.md\n", cfg.TargetDir)
	return res, nil
}

func readGoModule(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			old := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			if old == "" {
				return "", fmt.Errorf("empty module")
			}
			return old, nil
		}
	}
	return "", fmt.Errorf("no module")
}

func writeCombination(dest, id string, tokens map[string]string) ([]string, error) {
	prefix := "combinations/" + id
	var files []string
	err := fs.WalkDir(embedded.Combinations, prefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(path, prefix+"/")
		if rel == path || rel == "" {
			return fmt.Errorf("combination 文件路径异常: %s", path)
		}
		if err := safefs.RelOK(rel); err != nil {
			return err
		}
		data, err := embedded.Combinations.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		for k, v := range tokens {
			text = strings.ReplaceAll(text, k, v)
		}
		if err := safefs.WriteFile(dest, rel, []byte(text), 0o644); err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func gitInit(dir string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return err
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}
