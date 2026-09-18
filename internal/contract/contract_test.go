package contract

import "testing"

func TestParseRejectsUnknownTransform(t *testing.T) {
	t.Parallel()
	_, err := Parse([]byte(`
schema_version: 1
component: backend
payload:
  include: [cmd]
transforms:
  - kind: shell
    value_from: project_name
`))
	if err == nil {
		t.Fatal("expected unknown transform to fail")
	}
}

func TestParseRejectsPathTraversal(t *testing.T) {
	t.Parallel()
	_, err := Parse([]byte(`
schema_version: 1
component: backend
payload:
  include: ["../secret"]
`))
	if err == nil {
		t.Fatal("expected traversal include to fail")
	}
}

func TestParseDeployStackRequiresFrom(t *testing.T) {
	t.Parallel()
	_, err := Parse([]byte(`
schema_version: 1
component: backend
payload:
  include: [deploy]
transforms:
  - kind: deploy_stack
    path: deploy
    value_from: project_name
`))
	if err == nil {
		t.Fatal("expected deploy_stack without from to fail")
	}
}

func TestParseDeployStackOK(t *testing.T) {
	t.Parallel()
	c, err := Parse([]byte(`
schema_version: 1
component: backend
payload:
  include: [deploy]
transforms:
  - kind: deploy_stack
    path: deploy
    from: springhere
    value_from: project_name
`))
	if err != nil {
		t.Fatal(err)
	}
	if c.Transforms[0].From != "springhere" {
		t.Fatalf("from=%q", c.Transforms[0].From)
	}
}

func TestParseOK(t *testing.T) {
	t.Parallel()
	c, err := Parse([]byte(`
schema_version: 1
component: frontend
payload:
  include: [src, package.json]
transforms:
  - kind: json_string
    path: package.json
    key: name
    value_from: frontend_package_name
`))
	if err != nil {
		t.Fatal(err)
	}
	if !stringsHasPrefix(c.SHA256, "sha256:") {
		t.Fatalf("sha: %s", c.SHA256)
	}
}

func stringsHasPrefix(s, p string) bool {
	return len(s) >= len(p) && s[:len(p)] == p
}
