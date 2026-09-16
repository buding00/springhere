package safefs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/buding00/springhere/internal/clierr"
)

func RelOK(rel string) error {
	rel = filepath.ToSlash(rel)
	if rel == "" || rel == "." {
		return fmt.Errorf("空相对路径")
	}
	if filepath.IsAbs(rel) {
		return fmt.Errorf("拒绝绝对路径: %s", rel)
	}
	clean := filepath.ToSlash(filepath.Clean(rel))
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("拒绝路径穿越: %s", rel)
	}
	return nil
}

func DirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	if errors.Is(err, io.EOF) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return false, nil
}

func TargetWritable(target string) error {
	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return clierr.Conflict("目标已存在且不是目录: " + target)
	}
	empty, err := DirEmpty(target)
	if err != nil {
		return err
	}
	if !empty {
		return clierr.Conflict("目标目录非空，拒绝覆盖: " + target + "（换一个目录，或删掉后再生成）")
	}
	return nil
}

func Commit(staging, target string) error {
	if err := TargetWritable(target); err != nil {
		return err
	}
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(target); err == nil {
		if err := os.Remove(target); err != nil {
			return clierr.Conflict("无法替换空的目标目录: " + target)
		}
	}
	if err := os.Rename(staging, target); err != nil {
		return fmt.Errorf("无法将临时目录提交到 %s: %w", target, err)
	}
	return nil
}
