package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNormalizeTag(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"v0.1.0":   "v0.1.0",
		"0.1.0":    "v0.1.0",
		" v0.2.0 ": "v0.2.0",
		"":         "",
		"devel":    "",
		"(devel)":  "",
	}
	for in, want := range cases {
		if got := NormalizeTag(in); got != want {
			t.Fatalf("NormalizeTag(%q)=%q want %q", in, got, want)
		}
	}
}

func TestAssetName(t *testing.T) {
	t.Parallel()
	got := AssetName("v0.1.0", "darwin", "arm64")
	if got != "springhere_v0.1.0_darwin_arm64.tar.gz" {
		t.Fatal(got)
	}
	got = AssetName("0.1.0", "windows", "amd64")
	if got != "springhere_v0.1.0_windows_amd64.zip" {
		t.Fatal(got)
	}
}

func TestLatestStable(t *testing.T) {
	t.Parallel()
	releases := []Release{
		{Tag: "v0.1.0"},
		{Tag: "v0.3.0-rc.1", Pre: true},
		{Tag: "v0.2.1", Draft: true},
		{Tag: "v0.2.0"},
		{Tag: "nightly"},
		{Tag: "v0.0.9"},
	}
	got, err := LatestStable(releases)
	if err != nil {
		t.Fatal(err)
	}
	if got.Tag != "v0.2.0" {
		t.Fatalf("got %s", got.Tag)
	}
}

func TestLatestStableEmpty(t *testing.T) {
	t.Parallel()
	if _, err := LatestStable([]Release{{Tag: "v0.1.0-rc.1", Pre: true}}); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseChecksum(t *testing.T) {
	t.Parallel()
	name := "springhere_v0.1.0_linux_amd64.tar.gz"
	sum := strings.Repeat("ab", 32)
	body := fmt.Sprintf("%s  %s\n", sum, name)
	got, err := parseChecksum([]byte(body), name)
	if err != nil {
		t.Fatal(err)
	}
	if got != sum {
		t.Fatalf("got %s", got)
	}
	body = fmt.Sprintf("%s *%s\n", sum, name)
	got, err = parseChecksum([]byte(body), name)
	if err != nil {
		t.Fatal(err)
	}
	if got != sum {
		t.Fatalf("star prefix: %s", got)
	}
	if _, err := parseChecksum([]byte(body), "other.tar.gz"); err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestRunCheckAndInstall(t *testing.T) {
	goos, goarch := runtime.GOOS, runtime.GOARCH
	payload := []byte("new-springhere-binary")
	tag := "v0.2.0"
	asset := AssetName(tag, goos, goarch)
	archive := packArchive(t, goos, payload)
	sum := sha256.Sum256(archive)
	checksums := fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), asset)

	var base string
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/buding00/springhere/releases", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"tag_name":   "v0.1.0",
				"draft":      false,
				"prerelease": false,
				"html_url":   "https://github.com/buding00/springhere/releases/tag/v0.1.0",
				"assets":     []any{},
			},
			{
				"tag_name":   tag,
				"draft":      false,
				"prerelease": false,
				"html_url":   "https://github.com/buding00/springhere/releases/tag/" + tag,
				"assets": []map[string]any{
					{"name": asset, "browser_download_url": base + "/" + asset, "size": len(archive)},
					{"name": ChecksumFileName(), "browser_download_url": base + "/checksums.txt", "size": len(checksums)},
				},
			},
			{
				"tag_name":   "v0.3.0-rc.1",
				"draft":      false,
				"prerelease": true,
				"html_url":   "https://example.com/rc",
				"assets":     []any{},
			},
		})
	})
	mux.HandleFunc("/"+asset, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(archive) })
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(checksums)) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	base = srv.URL

	client := &Client{HTTP: srv.Client(), APIBase: srv.URL, Owner: DefaultOwner, Repo: DefaultRepo}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	stdout := &bytes.Buffer{}
	if err := Run(ctx, Options{
		Check:          true,
		CurrentVersion: "0.1.0",
		GOOS:           goos,
		GOARCH:         goarch,
		Stdout:         stdout,
		Client:         client,
		Executable:     filepath.Join(t.TempDir(), "springhere"),
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "可更新") || !strings.Contains(stdout.String(), tag) {
		t.Fatalf("check output: %s", stdout.String())
	}

	dest := filepath.Join(t.TempDir(), "springhere")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := Run(ctx, Options{
		Yes:            true,
		CurrentVersion: "0.1.0",
		GOOS:           goos,
		GOARCH:         goarch,
		Stdout:         stdout,
		Client:         client,
		Executable:     dest,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("installed %q", got)
	}
	if !strings.Contains(stdout.String(), "已安装 springhere 0.2.0") {
		t.Fatalf("install output: %s", stdout.String())
	}

	stdout.Reset()
	if err := Run(ctx, Options{
		CurrentVersion: "0.2.0",
		GOOS:           goos,
		GOARCH:         goarch,
		Stdout:         stdout,
		Client:         client,
		Executable:     dest,
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "已是最新稳定版") {
		t.Fatalf("up to date: %s", stdout.String())
	}
}

func TestRunChecksumMismatchKeepsOld(t *testing.T) {
	goos, goarch := runtime.GOOS, runtime.GOARCH
	payload := []byte("new-springhere-binary")
	tag := "v0.2.0"
	asset := AssetName(tag, goos, goarch)
	archive := packArchive(t, goos, payload)
	bad := strings.Repeat("00", 32)
	checksums := fmt.Sprintf("%s  %s\n", bad, asset)

	var base string
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/buding00/springhere/releases", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"tag_name": tag, "draft": false, "prerelease": false,
			"assets": []map[string]any{
				{"name": asset, "browser_download_url": base + "/" + asset},
				{"name": ChecksumFileName(), "browser_download_url": base + "/checksums.txt"},
			},
		}})
	})
	mux.HandleFunc("/"+asset, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(archive) })
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(checksums)) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	base = srv.URL

	dest := filepath.Join(t.TempDir(), "springhere")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := Run(context.Background(), Options{
		Yes:            true,
		CurrentVersion: "0.1.0",
		GOOS:           goos,
		GOARCH:         goarch,
		Stdout:         io.Discard,
		Client:         &Client{HTTP: srv.Client(), APIBase: srv.URL, Owner: DefaultOwner, Repo: DefaultRepo},
		Executable:     dest,
	})
	if err == nil {
		t.Fatal("expected checksum error")
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "old" {
		t.Fatalf("old binary was replaced: %q", got)
	}
}

func TestRunRequiresYes(t *testing.T) {
	tag := "v0.2.0"
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/buding00/springhere/releases", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{{
			"tag_name": tag, "draft": false, "prerelease": false,
			"assets": []any{},
		}})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	err := Run(context.Background(), Options{
		CurrentVersion: "0.1.0",
		Interactive:    false,
		Yes:            false,
		Stdout:         io.Discard,
		Client:         &Client{HTTP: srv.Client(), APIBase: srv.URL, Owner: DefaultOwner, Repo: DefaultRepo},
		Executable:     filepath.Join(t.TempDir(), "springhere"),
	})
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("got %v", err)
	}
}

func TestRunPinVersion(t *testing.T) {
	goos, goarch := runtime.GOOS, runtime.GOARCH
	payload := []byte("pinned")
	tag := "v0.1.0"
	asset := AssetName(tag, goos, goarch)
	archive := packArchive(t, goos, payload)
	sum := sha256.Sum256(archive)
	checksums := fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), asset)

	var base string
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/buding00/springhere/releases", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"tag_name": "v0.2.0", "draft": false, "prerelease": false,
				"assets": []any{},
			},
			{
				"tag_name": tag, "draft": false, "prerelease": false,
				"assets": []map[string]any{
					{"name": asset, "browser_download_url": base + "/" + asset},
					{"name": ChecksumFileName(), "browser_download_url": base + "/checksums.txt"},
				},
			},
		})
	})
	mux.HandleFunc("/"+asset, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(archive) })
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(checksums)) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	base = srv.URL

	dest := filepath.Join(t.TempDir(), "springhere")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Run(context.Background(), Options{
		Yes:            true,
		Version:        "0.1.0",
		CurrentVersion: "0.2.0",
		GOOS:           goos,
		GOARCH:         goarch,
		Stdout:         io.Discard,
		Client:         &Client{HTTP: srv.Client(), APIBase: srv.URL, Owner: DefaultOwner, Repo: DefaultRepo},
		Executable:     dest,
	}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "pinned" {
		t.Fatalf("got %q", got)
	}
}

func packArchive(t *testing.T, goos string, payload []byte) []byte {
	t.Helper()
	if goos == "windows" {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		w, err := zw.Create("springhere.exe")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(payload); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: "springhere", Mode: 0o755, Size: int64(len(payload))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
