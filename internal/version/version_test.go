package version

import "testing"

func TestDisplayAndTag(t *testing.T) {
	t.Parallel()
	if Display() == "" {
		t.Fatal("Display empty")
	}
	if Tag() != "v"+Display() {
		t.Fatalf("Tag=%q Display=%q", Tag(), Display())
	}
}
