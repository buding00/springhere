package selfupdate

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/buding00/springhere/internal/clierr"
)

func replaceExecutable(dest string, binary []byte) error {
	if dest == "" {
		return clierr.Source("无法确定当前 CLI 路径", nil)
	}
	abs, err := filepath.Abs(dest)
	if err != nil {
		return clierr.Source("无法解析当前 CLI 路径", err)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}

	dir := filepath.Dir(abs)
	tmp, err := os.CreateTemp(dir, ".springhere-update-*")
	if err != nil {
		return clierr.Conflict("无法在 " + dir + " 写入新版本（请确认对该目录有写权限）")
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(binary); err != nil {
		_ = tmp.Close()
		return clierr.Conflict("写入临时文件失败")
	}
	if err := tmp.Chmod(0o755); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		bak := abs + ".old"
		_ = os.Remove(bak)
		if _, err := os.Stat(abs); err == nil {
			if err := os.Rename(abs, bak); err != nil {
				return clierr.Conflict("无法替换当前 CLI: " + abs + "（请确认对该路径有写权限，或先退出正在运行的 springhere）")
			}
		}
		if err := os.Rename(tmpName, abs); err != nil {
			_ = os.Rename(bak, abs)
			return clierr.Conflict("无法安装新版本到 " + abs)
		}
		cleanup = false
		return nil
	}

	if err := os.Rename(tmpName, abs); err != nil {
		return clierr.Conflict("无法替换当前 CLI: " + abs + "（请确认对该路径有写权限）")
	}
	cleanup = false
	return nil
}
