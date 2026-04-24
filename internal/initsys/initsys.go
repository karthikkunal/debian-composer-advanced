// Package initsys provides init-system detection and unified service management
// across systemd, sysvinit, OpenRC, and runit.
//
// This is the single canonical package for all init system operations in Debian Composer.
// All service management flows through this package — no scattered systemctl/service/sv calls
// outside of initsys. This prevents fragmentation across init systems.
//
// Detection order:
//  1. /proc/1/comm (most reliable — name of PID 1's executable)
//  2. /run/systemd/private (systemd socket)
//  3. binary presence (openrc, sv, update-rc.d)
//
// Service operations per init system:
//   - systemd:  systemctl <cmd> <name>
//   - sysvinit: update-rc.d <name> <cmd>; service <name> <cmd>
//   - openrc:    rc-update <cmd> <name> <runlevel>; rc-service <name> <cmd>
//   - runit:    ln -sf /etc/sv/<name> <rundir>; sv <cmd> <name>
package initsys

import (
	"bufio"
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
		return Runit
	}
	if _, err := exec.LookPath("update-rc.d"); err == nil {
		return SysVInit
	}

	return Unknown
}

// SystemdFeatures enumerates which systemd features are available.
type SystemdFeatures struct {
	Systemd          bool // systemd is running
	SystemdJournal   bool // journald logging
	SystemdNetworkd  bool // networkd (systemd-networkd)
	SystemdResolved  bool // resolved (systemd-resolved)
	SystemdLogind    bool // logind (systemd-logind)
	SystemdTimedated bool // timedated (systemd-timedated)
}

// DetectSystemdFeatures probes which systemd features are available.
func DetectSystemdFeatures() SystemdFeatures {
	f := SystemdFeatures{Systemd: Detect() == Systemd}
	if !f.Systemd {
		return f
	}
	if _, err := exec.LookPath("journalctl"); err == nil {
		f.SystemdJournal = true
	}
	if _, err := os.Stat("/run/systemd/netif"); err == nil {
		f.SystemdNetworkd = true
	}
	if _, err := os.Stat("/run/systemd/resolve"); err == nil {
		f.SystemdResolved = true
	}
	out, _ := exec.Command("loginctl", "show-session", "self").Output()
	f.SystemdLogind = strings.Contains(string(out), "Leader=")
	if _, err := exec.LookPath("timedatectl"); err == nil {
		f.SystemdTimedated = true
	}
	return f
}

// HasFeature returns true if the named systemd feature is available.
func HasFeature(name string) bool {
	f := DetectSystemdFeatures()
	switch name {
	case "systemd", "systemd-journald", "journald":
		return f.SystemdJournal
	case "systemd-networkd", "networkd":
		return f.SystemdNetworkd
	case "systemd-resolved", "resolved":
		return f.SystemdResolved
	case "systemd-logind", "logind":
		return f.SystemdLogind
	case "systemd-timedated", "timedated":
		return f.SystemdTimedated
	default:
		return false
	}
}

// SystemdFeatureNames returns the list of available systemd feature names.
func SystemdFeatureNames() []string {
	f := DetectSystemdFeatures()
	var names []string
	if f.SystemdJournal {
		names = append(names, "journald")
	}
	if f.SystemdNetworkd {
		names = append(names, "networkd")
	}
	if f.SystemdResolved {
		names = append(names, "resolved")
	}
	if f.SystemdLogind {
		names = append(names, "logind")
	}
	if f.SystemdTimedated {
		names = append(names, "timedated")
	}
	return names
}

// ServiceStatus describes the state of a service.
type ServiceStatus struct {
	Name    string
	Active  bool
	Enabled bool
	Loaded  bool
}

// Status returns detailed status for a service across all init systems.
func Status(name string) (ServiceStatus, error) {
	st := ServiceStatus{Name: name}

	switch Detect() {
	case Systemd:
		out, _ := exec.Command("systemctl", "is-active", name).Output()
		st.Active = strings.TrimSpace(string(out)) == "active"
		st.Loaded, _ = unitExists(name)
		enab, _ := exec.Command("systemctl", "is-enabled", name).Output()
		st.Enabled = !strings.Contains(string(enab), "disabled")

	case OpenRC:
		st.Active = IsActive(name)
		out, _ := exec.Command("rc-update", "show").Output()
		st.Enabled = strings.Contains(string(out), name)

	case Runit:
		st.Active = IsActive(name)
		if fi, err := os.Lstat("/etc/runit/runsvdir/default/" + name); err == nil {
			st.Enabled = !fi.Mode().IsRegular()
		}

	default:
		st.Active = IsActive(name)
		for _, rl := range []string{"2", "3", "4", "5"} {
			if _, err := os.Lstat(fmt.Sprintf("/etc/rc%s.d/S??%s", rl, name)); err == nil {
				st.Enabled = true
				break
			}
		}
	}

	return st, nil
}

func unitExists(name string) (bool, error) {
	out, err := exec.Command("systemctl", "cat", name).Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(out), name+".service"), nil
}

// StartService starts a named service.
func StartService(name string) error {
	switch Detect() {
	case Systemd:
		return run("systemctl", "start", name)
	case OpenRC:
		return run("rc-service", name, "start")
	case Runit:
		return run("sv", "start", name)
	default:
		return run("service", name, "start")
	}
}

// StopService stops a named service.
func StopService(name string) error {
	switch Detect() {
	case Systemd:
		return run("systemctl", "stop", name)
	case OpenRC:
		return run("rc-service", name, "stop")
	case Runit:
		return run("sv", "stop", name)
	default:
		return run("service", name, "stop")
	}
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
		svdir := "/etc/sv/" + name
		if fi, err := os.Stat(svdir); err != nil || !fi.IsDir() {
			return fmt.Errorf("runit service %s: /etc/sv/%s is not a directory", name, name)
		}
		rundir := "/etc/runit/runsvdir/default"
		os.Remove(rundir + "/" + name)
		if err := os.Symlink(svdir, rundir+"/"+name); err != nil {
			return fmt.Errorf("runit enable %s: %w", name, err)
		}
		return nil
	default:
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
	case Runit:
		return os.Remove("/etc/runit/runsvdir/default/" + name)
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
		return exec.Command("service", name, "status").Run() == nil
	}
}

// ReloadDaemon reloads the init daemon configuration.
// No-op on non-systemd systems.
func ReloadDaemon() error {
	if Detect() == Systemd {
		return run("systemctl", "daemon-reload")
	}
	return nil
}

// ListServices returns all registered services (empty on non-systemd).
func ListServices(pattern string) ([]string, error) {
	switch Detect() {
	case Systemd:
		out, err := exec.Command("systemctl", "list-unit-files",
			"--type=service", "--no-pager", "--no-legend").Output()
		if err != nil {
			return nil, err
		}
		var names []string
		sc := bufio.NewScanner(strings.NewReader(string(out)))
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "UNIT") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 1 {
				continue
			}
			name := fields[0]
			if pattern != "" && !strings.Contains(name, pattern) {
				continue
			}
			names = append(names, name)
		}
		return names, nil
	default:
		entries, err := os.ReadDir("/etc/init.d")
		if err != nil {
			return nil, err
		}
		var names []string
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if pattern != "" && !strings.Contains(name, pattern) {
				continue
			}
			names = append(names, name)
		}
		return names, nil
	}
}

// ServiceCommand returns the command slice for a service operation.
func ServiceCommand(op string, name string) []string {
	switch Detect() {
	case Systemd:
		return []string{"systemctl", op, name}
	case OpenRC:
		return []string{"rc-service", name, op}
	case Runit:
		return []string{"sv", op, name}
	default:
		return []string{"service", name, op}
	}
}

// ServiceCommandWithUpdateRCD returns the command slice for update-rc.d-based operations.
func ServiceCommandWithUpdateRCD(op string, name string) []string {
	switch Detect() {
	case Systemd:
		return []string{"systemctl", op, name}
	case OpenRC:
		switch op {
		case "enable", "start":
			return []string{"rc-update", "add", name, "default"}
		case "disable", "stop":
			return []string{"rc-update", "del", name, "default"}
		default:
			return []string{"rc-service", name, op}
		}
	case Runit:
		return []string{"sv", op, name}
	default:
		if op == "enable" {
			return []string{"update-rc.d", name, "defaults"}
		}
		if op == "disable" {
			return []string{"update-rc.d", name, "remove"}
		}
		return []string{"service", name, op}
	}
}

// CommandString returns a human-readable command string for display.
func CommandString(args []string) string {
	return strings.Join(args, " ")
}

func run(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s failed: %w\nOutput: %s", name, strings.Join(args, " "), err, out)
	}
	return nil
}
