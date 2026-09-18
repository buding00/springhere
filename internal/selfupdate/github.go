package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/buding00/springhere/internal/clierr"
)

const (
	DefaultOwner = "buding00"
	DefaultRepo  = "springhere"
	defaultAPI   = "https://api.github.com"
	userAgent    = "springhere"
	maxBody      = 80 << 20
)

type Client struct {
	HTTP    *http.Client
	APIBase string
	Owner   string
	Repo    string
	Token   string
}

type Release struct {
	Tag     string
	Draft   bool
	Pre     bool
	HTMLURL string
	Assets  []Asset
}

type Asset struct {
	Name string
	URL  string
	Size int64
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 3 * time.Minute}
	}
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	return &Client{
		HTTP:    httpClient,
		APIBase: defaultAPI,
		Owner:   DefaultOwner,
		Repo:    DefaultRepo,
		Token:   token,
	}
}

func (c *Client) ListReleases(ctx context.Context) ([]Release, error) {
	c = c.ready()
	url := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=100", c.APIBase, c.Owner, c.Repo)
	body, err := c.get(ctx, url, true)
	if err != nil {
		return nil, err
	}
	var raw []struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
		HTMLURL    string `json:"html_url"`
		Assets     []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
			Size               int64  `json:"size"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, clierr.Source("无法解析 GitHub Release 列表", err)
	}
	out := make([]Release, 0, len(raw))
	for _, r := range raw {
		rel := Release{
			Tag:     r.TagName,
			Draft:   r.Draft,
			Pre:     r.Prerelease,
			HTMLURL: r.HTMLURL,
		}
		for _, a := range r.Assets {
			rel.Assets = append(rel.Assets, Asset{Name: a.Name, URL: a.BrowserDownloadURL, Size: a.Size})
		}
		out = append(out, rel)
	}
	return out, nil
}

func (c *Client) Download(ctx context.Context, url string) ([]byte, error) {
	c = c.ready()
	return c.get(ctx, url, false)
}

func (c *Client) ready() *Client {
	if c == nil {
		c = NewClient(nil)
	}
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 3 * time.Minute}
	}
	if c.APIBase == "" {
		c.APIBase = defaultAPI
	}
	if c.Owner == "" {
		c.Owner = DefaultOwner
	}
	if c.Repo == "" {
		c.Repo = DefaultRepo
	}
	return c
}

func (c *Client) get(ctx context.Context, url string, jsonAPI bool) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	if jsonAPI {
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, clierr.Source("请求 GitHub 失败。update 只从 GitHub Release 下载，请确认可以访问 github.com", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, clierr.Source("读取 GitHub 响应失败", err)
	}
	if int64(len(data)) > maxBody {
		return nil, clierr.Source("GitHub 响应过大，已拒绝", nil)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return data, nil
	case http.StatusNotFound:
		return nil, clierr.Source("未找到 GitHub Release。请确认 https://github.com/"+c.Owner+"/"+c.Repo+"/releases 已有稳定版本", nil)
	case http.StatusForbidden, http.StatusTooManyRequests:
		return nil, clierr.Source("GitHub API 被限流或拒绝访问，可设置环境变量 GITHUB_TOKEN 后重试", nil)
	default:
		msg := strings.TrimSpace(string(data))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return nil, clierr.Source(fmt.Sprintf("GitHub 返回 HTTP %d %s", resp.StatusCode, msg), nil)
	}
}
