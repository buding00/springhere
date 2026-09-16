package safefs

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/buding00/springhere/internal/clierr"
)

func CopyAllowlist(srcRoot, dstRoot string, includes []string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if IsExcluded(rel, d) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() && !CouldContainInclude(rel, includes) {
			return filepath.SkipDir
		}
		if !MatchInclude(rel, includes) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return clierr.Contract("拒绝复制符号链接: "+filepath.ToSlash(rel), nil)
		}
		dest := filepath.Join(dstRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0o755)
		}
		if err := copyFile(path, dest, info.Mode().Perm()); err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func WriteFile(root, rel string, data []byte, perm os.FileMode) error {
	if err := RelOK(rel); err != nil {
		return err
	}
	dest := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, data, perm)
}
