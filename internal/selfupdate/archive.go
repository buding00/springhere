package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/buding00/springhere/internal/clierr"
	"github.com/buding00/springhere/internal/safefs"
)

func parseChecksum(data []byte, filename string) (string, error) {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		sum, name, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		name = strings.TrimPrefix(name, "*")
		if name != filename {
			continue
		}
		sum = strings.ToLower(strings.TrimSpace(sum))
		if len(sum) != 64 {
			return "", clierr.Source("checksums.txt 中 "+filename+" 的 sha256 格式无效", nil)
		}
		if _, err := hex.DecodeString(sum); err != nil {
			return "", clierr.Source("checksums.txt 中 "+filename+" 的 sha256 格式无效", err)
		}
		return sum, nil
	}
	return "", clierr.Source("checksums.txt 中没有 "+filename, nil)
}

func verifySHA256(data []byte, want string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, want) {
		return clierr.Source(fmt.Sprintf("安装包校验失败（sha256 不匹配），已拒绝替换。got=%s want=%s", got, want), nil)
	}
	return nil
}

func extractBinary(archive []byte, goos, filename string) ([]byte, error) {
	if strings.HasSuffix(filename, ".zip") || goos == "windows" {
		return extractZip(archive)
	}
	return extractTarGz(archive)
}

func extractTarGz(archive []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, clierr.Source("无法解开 tar.gz", err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, clierr.Source("读取 tar 失败", err)
		}
		switch hdr.Typeflag {
		case tar.TypeXHeader, tar.TypeXGlobalHeader:
			continue
		case tar.TypeReg, tar.TypeRegA:
			name := path.Base(path.Clean("/" + strings.ReplaceAll(hdr.Name, "\\", "/")))
			if err := safefs.RelOK(hdr.Name); err != nil {
				return nil, clierr.Source("安装包含非法路径 "+hdr.Name, err)
			}
			if name != "springhere" {
				continue
			}
			data, err := io.ReadAll(io.LimitReader(tr, maxBody+1))
			if err != nil {
				return nil, clierr.Source("读取安装包中的二进制失败", err)
			}
			if int64(len(data)) > maxBody {
				return nil, clierr.Source("安装包中的二进制过大", nil)
			}
			if len(data) == 0 {
				return nil, clierr.Source("安装包中的 springhere 为空", nil)
			}
			return data, nil
		case tar.TypeSymlink, tar.TypeLink:
			return nil, clierr.Source("拒绝安装包中的链接: "+hdr.Name, nil)
		}
	}
	return nil, clierr.Source("安装包里没有 springhere 可执行文件", nil)
}

func extractZip(archive []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, clierr.Source("无法解开 zip", err)
	}
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "/") {
			continue
		}
		if err := safefs.RelOK(f.Name); err != nil {
			return nil, clierr.Source("安装包含非法路径 "+f.Name, err)
		}
		name := path.Base(path.Clean("/" + strings.ReplaceAll(f.Name, "\\", "/")))
		if name != "springhere.exe" && name != "springhere" {
			continue
		}
		if f.FileInfo().Mode()&os.ModeSymlink != 0 {
			return nil, clierr.Source("拒绝安装包中的符号链接: "+f.Name, nil)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, clierr.Source("读取 zip 中的二进制失败", err)
		}
		data, err := io.ReadAll(io.LimitReader(rc, maxBody+1))
		rc.Close()
		if err != nil {
			return nil, clierr.Source("读取 zip 中的二进制失败", err)
		}
		if int64(len(data)) > maxBody {
			return nil, clierr.Source("安装包中的二进制过大", nil)
		}
		if len(data) == 0 {
			return nil, clierr.Source("安装包中的 springhere 为空", nil)
		}
		return data, nil
	}
	return nil, clierr.Source("安装包里没有 springhere.exe 可执行文件", nil)
}
