package initializer

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func testdata(t *testing.T, elem ...string) string {
	t.Helper()
	p := filepath.Join(append([]string{"..", "..", "testdata"}, elem...)...)
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestRunLocalDryRunAndCommit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	out := &bytes.Buffer{}
	parent := t.TempDir()
	target := filepath.Join(parent, "demo")

	cfg := Config{
		ProjectName:         "demo",
		TargetDir:           target,
		Backend:             "gin",
		Frontend:            "react",
		Module:              "github.com/acme/demo",
		Source:              "auto",
		BackendTemplateDir:  testdata(t, "backend"),
		FrontendTemplateDir: testdata(t, "frontend"),
		DryRun:              true,
		NoGit:               true,
		Stdout:              out,
	}
	res, err := Run(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("dry-run must not create target")
	}
	joined := strings.Join(res.Files, "\n")
	if !strings.Contains(joined, "backend/go.mod") || !strings.Contains(joined, "frontend/package.json") {
		t.Fatalf("files=%v", res.Files)
	}

	cfg.DryRun = false
	if _, err := Run(ctx, cfg); err != nil {
		t.Fatal(err)
	}
	mod, err := os.ReadFile(filepath.Join(target, "backend", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mod), "module github.com/acme/demo") {
		t.Fatalf("go.mod: %s", mod)
	}
	if strings.Contains(string(mod), "example.com/source-backend") {
		t.Fatal("source module leftover in go.mod")
	}
	mainGo, err := os.ReadFile(filepath.Join(target, "backend", "cmd", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mainGo), `github.com/acme/demo/internal/pkg`) {
		t.Fatalf("imports not rewritten: %s", mainGo)
	}
	pkgJSON, err := os.ReadFile(filepath.Join(target, "frontend", "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(pkgJSON), `"name": "demo-admin"`) {
		t.Fatalf("package.json: %s", pkgJSON)
	}
	appYAML, err := os.ReadFile(filepath.Join(target, "backend", "application.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(appYAML), "name: demo") {
		t.Fatalf("application.yaml: %s", appYAML)
	}
	manifest, err := os.ReadFile(filepath.Join(target, ".springhere.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), "combination: gin-react-admin") {
		t.Fatalf("manifest: %s", manifest)
	}
	compose, err := os.ReadFile(filepath.Join(target, "backend", "deploy", "demo", "docker-compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(compose)
	if !strings.Contains(got, "redis:") || !strings.Contains(got, "postgres:") {
		t.Fatalf("compose missing postgres/redis: %s", got)
	}
	if !strings.Contains(got, "container_name: demo-postgres") || !strings.Contains(got, "container_name: demo-redis") {
		t.Fatalf("container names: %s", got)
	}
	if !strings.Contains(got, "POSTGRES_USER: springhere") {
		t.Fatalf("db user rewritten: %s", got)
	}
	if _, err := os.Stat(filepath.Join(target, "backend", "deploy", "demo", "nginx.conf")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "deployments")); !os.IsNotExist(err) {
		t.Fatal("root deployments/ must not exist")
	}
	if _, err := os.Stat(filepath.Join(target, "Makefile")); !os.IsNotExist(err) {
		t.Fatal("root Makefile must not exist")
	}
	if _, err := os.Stat(filepath.Join(target, "README.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, ".github")); !os.IsNotExist(err) {
		t.Fatal("root .github must not exist")
	}
	if _, err := os.Stat(filepath.Join(target, ".git")); !os.IsNotExist(err) {
		t.Fatal("root must not have .git")
	}
	if _, err := os.Stat(filepath.Join(target, "backend", ".git")); !os.IsNotExist(err) {
		t.Fatal("backend must not keep template .git when --no-git")
	}
	if !strings.Contains(joined, "backend/deploy/demo/docker-compose.yaml") {
		t.Fatalf("dry-run files missing renamed compose: %v", res.Files)
	}

	if _, err := Run(ctx, cfg); err == nil {
		t.Fatal("expected non-empty target to fail")
	}
}

func TestGitInitBackendAndFrontend(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	target := filepath.Join(t.TempDir(), "demo")
	_, err := Run(context.Background(), Config{
		ProjectName:         "demo",
		TargetDir:           target,
		Backend:             "gin",
		Frontend:            "react",
		Module:              "github.com/acme/demo",
		Source:              "auto",
		BackendTemplateDir:  testdata(t, "backend"),
		FrontendTemplateDir: testdata(t, "frontend"),
		Stdout:              &bytes.Buffer{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, ".git")); !os.IsNotExist(err) {
		t.Fatal("root must not have .git")
	}
	if st, err := os.Stat(filepath.Join(target, "backend", ".git")); err != nil || !st.IsDir() {
		t.Fatalf("backend .git: %v", err)
	}
	if st, err := os.Stat(filepath.Join(target, "frontend", ".git")); err != nil || !st.IsDir() {
		t.Fatalf("frontend .git: %v", err)
	}
}

func TestUnknownBackend(t *testing.T) {
	t.Parallel()
	_, err := Run(context.Background(), Config{
		ProjectName:         "demo",
		TargetDir:           filepath.Join(t.TempDir(), "demo"),
		Backend:             "axum",
		Frontend:            "react",
		Module:              "github.com/acme/demo",
		Source:              "github",
		BackendTemplateDir:  testdata(t, "backend"),
		FrontendTemplateDir: testdata(t, "frontend"),
		NoGit:               true,
		Stdout:              &bytes.Buffer{},
	})
	if err == nil {
		t.Fatal("expected unknown backend to fail")
	}
}
