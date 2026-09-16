package registry

import "testing"

func TestLoadEmbedded(t *testing.T) {
	t.Parallel()
	r, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Combination("gin", "react"); err != nil {
		t.Fatal(err)
	}
	gin := r.Backend("gin")
	if gin.Sources["gitee"].URL != "https://gitee.com/lovebuding/springhere-gin-server.git" {
		t.Fatalf("gitee backend url: %s", gin.Sources["gitee"].URL)
	}
	react := r.Frontend("react")
	if react.Sources["gitee"].URL != "https://gitee.com/lovebuding/springhere-react-admin.git" {
		t.Fatalf("gitee frontend url: %s", react.Sources["gitee"].URL)
	}
	if _, err := r.Combination("axum", "react"); err == nil {
		t.Fatal("expected unknown backend to fail")
	}
	if _, err := r.Combination("gin", "vue"); err == nil {
		t.Fatal("expected unknown frontend to fail")
	}
}

func TestCommitMismatchRejected(t *testing.T) {
	t.Parallel()
	data := []byte(`
schema_version: 1
backends:
  - id: gin
    name: Gin
    language: go
    sources:
      github: {url: https://example.com/gin.git, ref: main, commit: aaa}
      gitee:  {url: https://example.com/gin.git, ref: main, commit: bbb}
frontends:
  - id: react
    name: React
    language: typescript
    sources:
      github: {url: https://example.com/react.git, ref: main, commit: ""}
combinations:
  - id: gin-react-admin
    backend: gin
    frontend: react
    tested: true
`)
	if _, err := Parse(data); err == nil {
		t.Fatal("expected commit mismatch to fail")
	}
}

func TestUntestedCombinationRejected(t *testing.T) {
	t.Parallel()
	data := []byte(`
schema_version: 1
backends:
  - id: gin
    name: Gin
    language: go
    sources:
      github: {url: https://example.com/gin.git, ref: main, commit: ""}
frontends:
  - id: react
    name: React
    language: typescript
    sources:
      github: {url: https://example.com/react.git, ref: main, commit: ""}
combinations:
  - id: gin-react-admin
    backend: gin
    frontend: react
    tested: false
`)
	r, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Combination("gin", "react"); err == nil {
		t.Fatal("expected untested combination to fail")
	}
}
