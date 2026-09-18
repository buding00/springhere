package initializer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buding00/springhere/internal/contract"
)

func TestRewriteAppBrand(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	cfg := filepath.Join(root, "src", "config", "index.ts")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	src := []byte(`export const appConfig = {
  name: 'SpringHere',
  mark: 'S',
  documentTitle: {
    'zh-CN': 'SpringHere 管理服务',
    'en-US': 'SpringHere Admin',
  },
  footer: {
    copyright: {
      'zh-CN': '© 2026 SpringHere',
      'en-US': '© 2026 SpringHere',
    },
  },
}
`)
	if err := os.WriteFile(cfg, src, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("<title>SpringHere 管理服务</title>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("SpringHere React Admin\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := rewriteAppBrand(root, contract.Transform{
		Kind: "app_brand",
		Path: "src/config/index.ts",
		From: "SpringHere",
	}, "my-app")
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	body := string(got)
	if strings.Contains(body, "SpringHere") {
		t.Fatalf("brand leftover: %s", body)
	}
	if !strings.Contains(body, "name: 'my-app'") {
		t.Fatalf("name: %s", body)
	}
	if !strings.Contains(body, "mark: 'M'") {
		t.Fatalf("mark: %s", body)
	}
	if !strings.Contains(body, "'my-app 管理服务'") || !strings.Contains(body, "'my-app Admin'") {
		t.Fatalf("titles: %s", body)
	}
	if !strings.Contains(body, "'© 2026 my-app'") {
		t.Fatalf("copyright: %s", body)
	}
	html, err := os.ReadFile(filepath.Join(root, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "<title>my-app 管理服务</title>") {
		t.Fatalf("html: %s", html)
	}
	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(agents), "my-app React Admin") {
		t.Fatalf("agents: %s", agents)
	}
}

func TestBrandMark(t *testing.T) {
	t.Parallel()
	got, err := brandMark("my-app")
	if err != nil || got != "M" {
		t.Fatalf("got %q %v", got, err)
	}
	got, err = brandMark("order-system")
	if err != nil || got != "O" {
		t.Fatalf("got %q %v", got, err)
	}
}
