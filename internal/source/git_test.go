package source

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestFetchGitFromLocalRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "hello.txt"), "hi\n")
	mustWrite(t, filepath.Join(src, ".springhere-template.yaml"), "schema_version: 1\n")
	runGit(t, src, "init", "-b", "main")
	runGit(t, src, "config", "user.email", "test@example.com")
	runGit(t, src, "config", "user.name", "test")
	runGit(t, src, "add", ".")
	runGit(t, src, "commit", "-m", "init")
	sha := runGit(t, src, "rev-parse", "HEAD")

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	fetched, err := FetchGit(ctx, "github", src, "main", sha)
	if err != nil {
		t.Fatal(err)
	}
	defer fetched.Cleanup()
	if fetched.Commit != sha {
		t.Fatalf("commit %s want %s", fetched.Commit, sha)
	}
	body, err := os.ReadFile(filepath.Join(fetched.Dir, "hello.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "hi\n" {
		t.Fatalf("body=%q", body)
	}

	if _, err := FetchGit(ctx, "github", src, "main", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); err == nil {
		t.Fatal("expected commit mismatch to fail")
	}
}

func TestPickMirrorExplicit(t *testing.T) {
	t.Parallel()
	got, _ := PickMirror("gitee", nil)
	if got != "gitee" {
		t.Fatalf("got %s", got)
	}
	got, _ = PickMirror("github", nil)
	if got != "github" {
		t.Fatalf("got %s", got)
	}
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s (%v)", args, out, err)
	}
	return string(trimNL(out))
}

func trimNL(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return b
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
