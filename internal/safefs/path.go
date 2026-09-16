package safefs

import (
	"io/fs"
	"path"
	"strings"
)

var excludedNames = map[string]struct{}{
	".git":         {},
	".idea":        {},
	".DS_Store":    {},
	"node_modules": {},
	".pnpm-store":  {},
	"dist":         {},
	"dist-ssr":     {},
	"bin":          {},
	"tmp":          {},
	"coverage.out": {},
}

func IsExcluded(rel string, d fs.DirEntry) bool {
	rel = path.Clean("/" + filepathToSlash(rel))
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" || rel == "." {
		return false
	}
	base := path.Base(rel)
	if _, ok := excludedNames[base]; ok {
		return true
	}
	if strings.HasSuffix(base, ".local") {
		return true
	}
	if d != nil && !d.IsDir() && strings.HasPrefix(base, ".env") && base != ".env.example" {
		return true
	}
	return false
}

func MatchInclude(rel string, includes []string) bool {
	rel = filepathToSlash(rel)
	if rel == "." || rel == "" {
		return false
	}
	for _, inc := range includes {
		inc = strings.TrimSuffix(filepathToSlash(inc), "/")
		if rel == inc || strings.HasPrefix(rel, inc+"/") {
			return true
		}
	}
	return false
}

func CouldContainInclude(rel string, includes []string) bool {
	rel = filepathToSlash(rel)
	if rel == "." || rel == "" {
		return true
	}
	for _, inc := range includes {
		inc = strings.TrimSuffix(filepathToSlash(inc), "/")
		if inc == rel || strings.HasPrefix(inc, rel+"/") || strings.HasPrefix(rel, inc+"/") {
			return true
		}
	}
	return false
}

func filepathToSlash(p string) string {
	return path.Clean(strings.ReplaceAll(p, "\\", "/"))
}
