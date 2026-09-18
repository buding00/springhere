package initializer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buding00/springhere/internal/contract"
)

func TestRewriteDeployStack(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := filepath.Join(root, "deploy", "springhere")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	compose := []byte(`services:
  postgres:
    container_name: springhere-postgres
    environment:
      POSTGRES_USER: springhere
  redis:
    container_name: springhere-redis
`)
	if err := os.WriteFile(filepath.Join(src, "docker-compose.yaml"), compose, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "nginx.conf"), []byte("server {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := rewriteDeployStack(root, contract.Transform{
		Kind: "deploy_stack",
		Path: "deploy",
		From: "springhere",
	}, "demo")
	if err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(root, "deploy", "demo")
	body, err := os.ReadFile(filepath.Join(dst, "docker-compose.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)
	if !strings.Contains(got, "container_name: demo-postgres") || !strings.Contains(got, "container_name: demo-redis") {
		t.Fatalf("container names: %s", got)
	}
	if !strings.Contains(got, "POSTGRES_USER: springhere") {
		t.Fatalf("db user rewritten: %s", got)
	}
	if _, err := os.Stat(filepath.Join(dst, "nginx.conf")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "deploy", "springhere")); !os.IsNotExist(err) {
		t.Fatal("old deploy dir still exists")
	}
}
