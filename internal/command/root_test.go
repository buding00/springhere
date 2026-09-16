package command

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelp(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd := NewRoot(bytes.NewReader(nil), stdout, stderr)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, "springhere") || !strings.Contains(out, "new") {
		t.Fatalf("help: %s", out)
	}
}

func TestNewHelp(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	cmd := NewRoot(bytes.NewReader(nil), stdout, &bytes.Buffer{})
	cmd.SetArgs([]string{"new", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, "--source") || !strings.Contains(out, "gitee") || !strings.Contains(out, "github") {
		t.Fatalf("new help: %s", out)
	}
}

func TestNewRequiresYesWithoutTTY(t *testing.T) {
	t.Parallel()
	cmd := NewRoot(bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{})
	cmd.SetArgs([]string{"new", "demo", "--module", "github.com/acme/demo"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected non-tty without --yes to fail")
	}
}

func TestNewUnknownBackend(t *testing.T) {
	t.Parallel()
	cmd := NewRoot(bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{})
	cmd.SetArgs([]string{"new", "demo", "--yes", "--backend", "axum", "--module", "github.com/acme/demo"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected unknown backend to fail")
	}
}
