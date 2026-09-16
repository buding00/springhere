package embedded

import "embed"

//go:embed components.yaml
var ComponentsYAML []byte

//go:embed all:combinations
var Combinations embed.FS
