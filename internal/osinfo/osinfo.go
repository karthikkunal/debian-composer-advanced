package osinfo

import (
	"bufio"
	"os"
	"strings"
	"sync"
)

// OSInfo holds information parsed from /etc/os-release
type OSInfo struct {
	ID        string // e.g. "debian", "devuan", "ubuntu"
	IDLike    string // e.g. "debian" (for Devuan)
	Name      string // e.g. "Devuan GNU/Linux"
	Version   string // e.g. "5 (daedalus)"
	VersionID string // e.g. "5"
}

var (
	cached *OSInfo
	once   sync.Once
)

// Get returns OS information, reading /etc/os-release once and caching the result.
func Get() *OSInfo {
	once.Do(func() {
		cached = parse("/etc/os-release")
	})
	return cached
}

func parse(path string) *OSInfo {
	info := &OSInfo{}
	f, err := os.Open(path)
	if err != nil {
		return info
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		key, val, _ := strings.Cut(line, "=")
		val = strings.Trim(val, `"'`)
		switch key {
		case "ID":
			info.ID = strings.ToLower(val)
		case "ID_LIKE":
			info.IDLike = strings.ToLower(val)
		case "NAME":
			info.Name = val
		case "VERSION":
			info.Version = val
		case "VERSION_ID":
			info.VersionID = val
		}
	}
	return info
}

// IsDevuan returns true when running on Devuan.
func IsDevuan() bool {
	return Get().ID == "devuan"
}

// IsDebianLike returns true when the OS is Debian or any Debian derivative
// (Ubuntu, Devuan, etc.).
func IsDebianLike() bool {
	info := Get()
	if info.ID == "debian" {
		return true
	}
	for _, part := range strings.Fields(info.IDLike) {
		if part == "debian" {
			return true
		}
	}
	return false
}
