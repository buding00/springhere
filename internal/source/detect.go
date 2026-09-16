package source

import (
	"net"
	"time"
)

type Detector interface {
	Latency(hostport string) (time.Duration, error)
}

type TCPDetector struct {
	Timeout time.Duration
}

func (d TCPDetector) Latency(hostport string) (time.Duration, error) {
	timeout := d.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	start := time.Now()
	conn, err := net.DialTimeout("tcp", hostport, timeout)
	if err != nil {
		return 0, err
	}
	_ = conn.Close()
	return time.Since(start), nil
}

func PickMirror(kind string, detect Detector) (string, string) {
	kind = normalizeKind(kind)
	if kind == "github" || kind == "gitee" {
		return kind, ""
	}
	if detect == nil {
		detect = TCPDetector{}
	}
	gh, ghErr := detect.Latency("github.com:443")
	ge, geErr := detect.Latency("gitee.com:443")
	switch {
	case ghErr != nil && geErr != nil:
		return "github", "GitHub 与 Gitee 均不可达，将先尝试 GitHub"
	case geErr == nil && (ghErr != nil || ge < gh):
		return "gitee", "已选择延迟更低的 Gitee 镜像"
	default:
		return "github", "已选择延迟更低的 GitHub 镜像"
	}
}

func normalizeKind(kind string) string {
	switch kind {
	case "gitee":
		return "gitee"
	case "github":
		return "github"
	default:
		return "auto"
	}
}
