package ip2regionhelper

import (
	"fmt"
	"gin-fast/app/global/app"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

const defaultIPv4XDBPath = "resource/ip2region/ip2region.xdb"

type resolver struct {
	mu        sync.RWMutex
	version   *xdb.Version
	content   []byte
	source    string
	available bool
}

var defaultResolver = &resolver{}

func Init() error {
	return defaultResolver.reload()
}

func Reload() error {
	return defaultResolver.reload()
}

func Lookup(ip string) string {
	return defaultResolver.lookup(ip)
}

func (r *resolver) reload() error {
	path := resolveIPv4XDBPath()
	if path == "" {
		r.mu.Lock()
		r.version = nil
		r.content = nil
		r.source = ""
		r.available = false
		r.mu.Unlock()
		return fmt.Errorf("ip2region xdb path is empty")
	}

	content, err := xdb.LoadContentFromFile(path)
	if err != nil {
		r.mu.Lock()
		r.version = nil
		r.content = nil
		r.source = ""
		r.available = false
		r.mu.Unlock()
		return err
	}

	header, err := xdb.LoadHeaderFromBuff(content)
	if err != nil {
		return err
	}

	version, err := xdb.VersionFromHeader(header)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.version = version
	r.content = content
	r.source = path
	r.available = true
	r.mu.Unlock()
	return nil
}

func (r *resolver) lookup(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	if parsed.IsLoopback() {
		return "本机地址"
	}
	if parsed.IsPrivate() {
		return "内网IP"
	}

	r.mu.RLock()
	version := r.version
	content := r.content
	available := r.available
	r.mu.RUnlock()
	if !available || version == nil || len(content) == 0 {
		return ""
	}

	searcher, err := xdb.NewWithBuffer(version, content)
	if err != nil {
		return ""
	}
	region, err := searcher.Search(ip)
	searcher.Close()
	if err != nil {
		return ""
	}
	return formatRegion(region)
}

func resolveIPv4XDBPath() string {
	configuredPath := strings.TrimSpace(app.ConfigYml.GetString("customer.ip2region.xdb_path"))
	if configuredPath == "" {
		configuredPath = defaultIPv4XDBPath
	}
	if filepath.IsAbs(configuredPath) {
		return configuredPath
	}
	if app.BasePath == "" {
		return configuredPath
	}
	return filepath.Join(app.BasePath, filepath.FromSlash(configuredPath))
}

func formatRegion(region string) string {
	parts := strings.Split(strings.TrimSpace(region), "|")
	if len(parts) == 0 {
		return ""
	}

	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "0" {
			continue
		}
		if !contains(result, part) {
			result = append(result, part)
		}
	}
	return strings.Join(result, " ")
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func XDBExists() bool {
	path := resolveIPv4XDBPath()
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
