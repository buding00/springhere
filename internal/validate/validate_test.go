package validate

import "testing"

func TestProjectName(t *testing.T) {
	t.Parallel()
	ok := []string{"app", "order-system", "a1", "gin"}
	for _, n := range ok {
		if err := ProjectName(n); err != nil {
			t.Fatalf("%s: %v", n, err)
		}
	}
	bad := []string{"", "App", "-x", "x-", "x--y", "../x", "a/b", "1abc", "A_b"}
	for _, n := range bad {
		if err := ProjectName(n); err == nil {
			t.Fatalf("expected error for %q", n)
		}
	}
}

func TestGoModule(t *testing.T) {
	t.Parallel()
	if err := GoModule("github.com/acme/order-system"); err != nil {
		t.Fatal(err)
	}
	if err := GoModule(""); err == nil {
		t.Fatal("expected empty module to fail")
	}
	if err := GoModule("not a module"); err == nil {
		t.Fatal("expected invalid module to fail")
	}
}

func TestSourceKind(t *testing.T) {
	t.Parallel()
	if err := SourceKind("github"); err != nil {
		t.Fatal(err)
	}
	if err := SourceKind("gitlab"); err == nil {
		t.Fatal("expected unknown source to fail")
	}
}

func TestNPMPackageName(t *testing.T) {
	t.Parallel()
	if got := NPMPackageName("order-system"); got != "order-system-admin" {
		t.Fatalf("got %s", got)
	}
	if got := NPMPackageName("demo-admin"); got != "demo-admin" {
		t.Fatalf("got %s", got)
	}
}
