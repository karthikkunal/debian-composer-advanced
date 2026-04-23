package service

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/debian-composer/debian-composer-go/internal/initsys"
	"github.com/taigrr/systemctl"
)

// Service represents a systemd service
type Service struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Enabled bool   `json:"enabled"`
	Failed  bool   `json:"failed"`
}

// ServiceAction represents an action to perform on a service
type ServiceAction struct {
	Name    string `json:"name"`
	Action  string `json:"action"` // enable, disable, start, stop, restart
	Enabled bool   `json:"enabled,omitempty"`
}

// Manager handles systemd service operations
type Manager struct {
	dryRun bool
	opts   systemctl.Options
}

// New creates a new service manager
func New(dryRun bool) *Manager {
	return &Manager{
		dryRun: dryRun,
		opts: systemctl.Options{
			UserMode: false,
		},
	}
}

// GetStatus returns the current status of a service
func (m *Manager) GetStatus(name string) (*Service, error) {
	if initsys.Detect() != initsys.Systemd {
		active := initsys.IsActive(name)
		return &Service{Name: name, Active: active}, nil
	}

	ctx := context.Background()

	active, err := systemctl.IsActive(ctx, name, m.opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get status of %s: %w", name, err)
	}

	enabled, _ := systemctl.IsEnabled(ctx, name, m.opts)
	failed, _ := systemctl.IsFailed(ctx, name, m.opts)

	return &Service{
		Name:    name,
		Active:  active,
		Enabled: enabled,
		Failed:  failed,
	}, nil
}

// Start starts a service
func (m *Manager) Start(name string) error {
	if m.dryRun {
		return nil
	}
	if initsys.Detect() != initsys.Systemd {
		out, err := exec.Command("service", name, "start").CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to start %s: %w\nOutput: %s", name, err, out)
		}
		return nil
	}
	ctx := context.Background()
	if err := systemctl.Start(ctx, name, m.opts); err != nil {
		return fmt.Errorf("failed to start %s: %w", name, err)
	}
	return nil
}

// Stop stops a service
func (m *Manager) Stop(name string) error {
	if m.dryRun {
		return nil
	}
	if initsys.Detect() != initsys.Systemd {
		out, err := exec.Command("service", name, "stop").CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to stop %s: %w\nOutput: %s", name, err, out)
		}
		return nil
	}
	ctx := context.Background()
	if err := systemctl.Stop(ctx, name, m.opts); err != nil {
		return fmt.Errorf("failed to stop %s: %w", name, err)
	}
	return nil
}

// Enable enables a service
func (m *Manager) Enable(name string) error {
	if m.dryRun {
		return nil
	}
	if initsys.Detect() != initsys.Systemd {
		return initsys.EnableService(name)
	}
	ctx := context.Background()
	if err := systemctl.Enable(ctx, name, m.opts); err != nil {
		return fmt.Errorf("failed to enable %s: %w", name, err)
	}
	return nil
}

// Disable disables a service
func (m *Manager) Disable(name string) error {
	if m.dryRun {
		return nil
	}
	if initsys.Detect() != initsys.Systemd {
		return initsys.DisableService(name)
	}
	ctx := context.Background()
	if err := systemctl.Disable(ctx, name, m.opts); err != nil {
		return fmt.Errorf("failed to disable %s: %w", name, err)
	}
	return nil
}

// Restart restarts a service
func (m *Manager) Restart(name string) error {
	if m.dryRun {
		return nil
	}
	if initsys.Detect() != initsys.Systemd {
		return initsys.RestartService(name)
	}
	ctx := context.Background()
	if err := systemctl.Restart(ctx, name, m.opts); err != nil {
		return fmt.Errorf("failed to restart %s: %w", name, err)
	}
	return nil
}

// Reload reloads the init daemon (no-op on non-systemd systems).
func (m *Manager) Reload() error {
	if m.dryRun {
		return nil
	}
	if initsys.Detect() != initsys.Systemd {
		// sysvinit/openrc have no equivalent global reload; silently skip
		return nil
	}
	ctx := context.Background()
	if err := systemctl.DaemonReload(ctx, m.opts); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}
	return nil
}

// ExecuteActions performs a list of service actions
func (m *Manager) ExecuteActions(actions []ServiceAction) error {
	for _, action := range actions {
		var err error
		switch strings.ToLower(action.Action) {
		case "enable":
			err = m.Enable(action.Name)
		case "disable":
			err = m.Disable(action.Name)
		case "start":
			err = m.Start(action.Name)
		case "stop":
			err = m.Stop(action.Name)
		case "restart":
			err = m.Restart(action.Name)
		default:
			err = fmt.Errorf("unknown action: %s", action.Action)
		}
		if err != nil {
			return fmt.Errorf("service %s: %w", action.Name, err)
		}
	}
	return nil
}

// ListServices returns all services matching a pattern.
// On non-systemd systems an empty list is returned (enumeration is init-specific).
func ListServices(pattern string) ([]Service, error) {
	if initsys.Detect() != initsys.Systemd {
		return nil, nil
	}
	ctx := context.Background()
	opts := systemctl.Options{UserMode: false}

	units, err := systemctl.GetUnits(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list units: %w", err)
	}

	var services []Service
	for _, unit := range units {
		if pattern != "" && !strings.Contains(unit.Name, pattern) {
			continue
		}
		if strings.HasSuffix(unit.Name, ".service") {
			enabled, _ := systemctl.IsEnabled(ctx, unit.Name, opts)
			services = append(services, Service{
				Name:    unit.Name,
				Active:  unit.Active == "active",
				Enabled: enabled,
			})
		}
	}
	return services, nil
}

// CheckService checks if a service exists and returns its status
func CheckService(name string) (exists bool, active bool, err error) {
	if initsys.Detect() != initsys.Systemd {
		active = initsys.IsActive(name)
		return active, active, nil
	}
	ctx := context.Background()
	opts := systemctl.Options{UserMode: false}

	active, err = systemctl.IsActive(ctx, name, opts)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "No such") {
			return false, false, nil
		}
		return false, false, err
	}

	return true, active, nil
}

// EnableServiceUnit enables a systemd service unit file
func EnableServiceUnit(unitFile string) error {
	cmd := exec.Command("sudo", "systemctl", "enable", unitFile)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable %s: %w", unitFile, err)
	}
	return nil
}
