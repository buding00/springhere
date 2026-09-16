package safefs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyAllowlist(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "cmd", "main.go"), "package main\n")
	mustWrite(t, filepath.Join(src, "secret.txt"), "nope")
	mustWrite(t, filepath.Join(src, ".env"), "SECRET=1")
	if err := os.MkdirAll(filepath.Join(src, "node_modules", "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(src, "node_modules", "x", "a.js"), "x")

	dst := t.TempDir()
	files, err := CopyAllowlist(src, dst, []string{"cmd"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "cmd/main.go" {
		t.Fatalf("files=%v", files)
	}
	if _, err := os.Stat(filepath.Join(dst, "secret.txt")); !os.IsNotExist(err) {
		t.Fatal("secret.txt should not be copied")
	}
}

func TestRejectSymlink(t *testing.T) {
	t.Parallel()
	src := t.TempDir()
	mustWrite(t, filepath.Join(src, "real.go"), "package p\n")
	if err := os.Symlink("real.go", filepath.Join(src, "link.go")); err != nil {
		t.Skip("symlink not supported")
	}
	_, err := CopyAllowlist(src, t.TempDir(), []string{"link.go"})
	if err == nil {
		t.Fatal("expected symlink reject")
	}
}

func TestTargetWritable(t *testing.T) {
	t.Parallel()
	empty := t.TempDir()
	if err := TargetWritable(empty); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "f"), "x")
	if err := TargetWritable(dir); err == nil {
		t.Fatal("expected non-empty to fail")
	}
}

func TestRelOK(t *testing.T) {
	t.Parallel()
	if err := RelOK("../x"); err == nil {
		t.Fatal("expected traversal fail")
	}
	if err := RelOK("backend/go.mod"); err != nil {
		t.Fatal(err)
	}
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
