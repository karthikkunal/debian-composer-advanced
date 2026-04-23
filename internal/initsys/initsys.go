// Package initsys provides init-system detection and service management that
// works across systemd (Debian default) and sysvinit/OpenRC (Devuan default).
package initsys

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// InitSystem identifies the running init system.
type InitSystem int

const (
	Unknown InitSystem = iota
	Systemd
	SysVInit
	OpenRC
	Runit
)

func (i InitSystem) String() string {
	switch i {
	case Systemd:
		return "systemd"
	case SysVInit:
		return "sysvinit"
	case OpenRC:
		return "openrc"
	case Runit:
		return "runit"
	default:
		return "unknown"
	}
}

var (
	detected   InitSystem
	detectOnce sync.Once
)

// Detect returns the init system currently running as PID 1.
// The result is cached after the first call.
func Detect() InitSystem {
	detectOnce.Do(func() {
		detected = detect()
	})
	return detected
}

func detect() InitSystem {
	// Most reliable: read /proc/1/comm (the name of PID 1's executable)
	if comm, err := os.ReadFile("/proc/1/comm"); err == nil {
		name := strings.TrimSpace(string(comm))
		switch name {
		case "systemd":
			return Systemd
		case "openrc-init", "openrc":
			return OpenRC
		case "runit", "runit-init":
			return Runit
		}
	}

	// Fallback: check well-known paths / binaries
	if _, err := os.Stat("/run/systemd/private"); err == nil {
		return Systemd
	}
	if _, err := exec.LookPath("openrc"); err == nil {
		return OpenRC
	}
	if _, err := exec.LookPath("sv"); err == nil {
		// sv is runit's service manager
		return Runit
	}
	if _, err := exec.LookPath("update-rc.d"); err == nil {
		return SysVInit
	}

	return Unknown
}

// RestartService restarts a named service using the appropriate init tool.
func RestartService(name string) error {
	switch Detect() {
	case Systemd:
		return run("systemctl", "restart", name)
	case OpenRC:
		return run("rc-service", name, "restart")
	case Runit:
		return run("sv", "restart", name)
	default:
		// sysvinit / unknown
		if err := run("service", name, "restart"); err != nil {
			return run("/etc/init.d/"+name, "restart")
		}
		return nil
	}
}

// EnableService enables a named service to start at boot.
func EnableService(name string) error {
	switch Detect() {
	case Systemd:
		return run("systemctl", "enable", name)
	case OpenRC:
		return run("rc-update", "add", name, "default")
	case Runit:
		return fmt.Errorf("runit service enable not yet supported: create symlink manually in /etc/runit/runsvdir/default/%s", name)
	default:
		// sysvinit
		return run("update-rc.d", name, "defaults")
	}
}

// DisableService disables a named service from starting at boot.
func DisableService(name string) error {
	switch Detect() {
	case Systemd:
		return run("systemctl", "disable", name)
	case OpenRC:
		return run("rc-update", "del", name, "default")
	default:
		return run("update-rc.d", name, "remove")
	}
}

// IsActive returns true if the named service is currently running.
func IsActive(name string) bool {
	switch Detect() {
	case Systemd:
		out, err := exec.Command("systemctl", "is-active", name).Output()
		return err == nil && strings.TrimSpace(string(out)) == "active"
	case OpenRC:
		return exec.Command("rc-service", name, "status").Run() == nil
	case Runit:
		out, err := exec.Command("sv", "status", name).Output()
		return err == nil && strings.HasPrefix(string(out), "run:")
	default:
		// sysvinit: exit 0 = running
		return exec.Command("service", name, "status").Run() == nil
	}
}

func run(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s failed: %w\nOutput: %s", name, strings.Join(args, " "), err, out)
	}
	return nil
}
