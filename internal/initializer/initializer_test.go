package initializer

import (
	"bytes"
	"context"
	"os"
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
	compose, err := os.ReadFile(filepath.Join(target, "deployments", "docker-compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(compose), "redis:") || !strings.Contains(string(compose), "postgres:") {
		t.Fatalf("compose missing postgres/redis: %s", compose)
	}
	if _, err := os.Stat(filepath.Join(target, "backend", ".git")); !os.IsNotExist(err) {
		t.Fatal("backend must not keep template .git")
	}

	if _, err := Run(ctx, cfg); err == nil {
		t.Fatal("expected non-empty target to fail")
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
