package conditions

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/expr-lang/expr"
)

// Context provides variables and functions for condition evaluation
type Context struct {
	Hardware    *HardwareEnv
	Environment *Environment
	State       StateChecker
	Variables   map[string]interface{}
}

// HardwareEnv provides hardware information to conditions
type HardwareEnv struct {
	Arch              string `expr:"arch"`
	CPUCores          int    `expr:"cpu_cores"`
	CPUThreads        int    `expr:"cpu_threads"`
	CPUModel          string `expr:"cpu_model"`
	RAMMB             int    `expr:"ram_mb"`
	RAMGB             int    `expr:"ram_gb"`
	DiskGB            int    `expr:"disk_gb"`
	HasGPU            bool   `expr:"has_gpu"`
	HasNVIDIA         bool   `expr:"has_nvidia"`
	HasAMD            bool   `expr:"has_amd"`
	HasIntel          bool   `expr:"has_intel"`
	HasVirtualization bool   `expr:"has_virtualization"`
}

// Environment provides system environment information
type Environment struct {
	User        string `expr:"user"`
	IsRoot      bool   `expr:"is_root"`
	HasDisplay  bool   `expr:"has_display"`
	DisplayType string `expr:"display_type"` // wayland, x11, none
	DesktopEnv  string `expr:"desktop_env"`  // gnome, kde, xfce, etc.
	Hostname    string `expr:"hostname"`
	HomeDir     string `expr:"home_dir"`
}

// StateChecker interface for checking installation state
type StateChecker interface {
	IsRecipeInstalled(name string) bool
	IsPackageInstalled(name string) bool
	IsServiceRunning(name string) bool
	FileExists(path string) bool
	CommandExists(name string) bool
}

// DefaultStateChecker provides default state checking
type DefaultStateChecker struct{}

func (d *DefaultStateChecker) IsRecipeInstalled(name string) bool {
	// Check state database
	return false
}

func (d *DefaultStateChecker) IsPackageInstalled(name string) bool {
	out, err := exec.Command("dpkg", "-l", name).Output()
	return err == nil && strings.Contains(string(out), "ii")
}

func (d *DefaultStateChecker) IsServiceRunning(name string) bool {
	out, err := exec.Command("systemctl", "is-active", name).Output()
	return err == nil && strings.TrimSpace(string(out)) == "active"
}

func (d *DefaultStateChecker) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (d *DefaultStateChecker) CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// Evaluator evaluates conditions using expr-lang/expr
type Evaluator struct {
	ctx *Context
}

// NewEvaluator creates a new condition evaluator
func NewEvaluator() *Evaluator {
	return &Evaluator{
		ctx: &Context{
			Hardware:    &HardwareEnv{},
			Environment: &Environment{},
			State:       &DefaultStateChecker{},
			Variables:   make(map[string]interface{}),
		},
	}
}

// SetHardware sets hardware information
func (e *Evaluator) SetHardware(hw *HardwareEnv) {
	e.ctx.Hardware = hw
}

// SetEnvironment sets environment information
func (e *Evaluator) SetEnvironment(env *Environment) {
	e.ctx.Environment = env
}

// SetStateChecker sets a custom state checker
func (e *Evaluator) SetStateChecker(sc StateChecker) {
	e.ctx.State = sc
}

// SetVariable sets a recipe variable
func (e *Evaluator) SetVariable(name string, value interface{}) {
	e.ctx.Variables[name] = value
}

// SetVariables sets multiple recipe variables
func (e *Evaluator) SetVariables(vars map[string]interface{}) {
	for k, v := range vars {
		e.ctx.Variables[k] = v
	}
}

// Eval evaluates a condition expression
func (e *Evaluator) Eval(condition string) (bool, error) {
	if condition == "" {
		return true, nil
	}

	env := e.buildEnv()

	// Compile with type checking
	program, err := expr.Compile(condition, expr.Env(env), expr.AsBool())
	if err != nil {
		return false, fmt.Errorf("compile condition '%s': %w", condition, err)
	}

	// Run the program
	result, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("eval condition '%s': %w", condition, err)
	}

	boolResult, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("condition '%s' did not return bool", condition)
	}

	return boolResult, nil
}

// EvalWithVars evaluates a condition with additional variables
func (e *Evaluator) EvalWithVars(condition string, vars map[string]interface{}) (bool, error) {
	// Save and restore variables
	saved := make(map[string]interface{})
	for k, v := range e.ctx.Variables {
		saved[k] = v
	}
	for k, v := range vars {
		e.ctx.Variables[k] = v
	}
	defer func() {
		e.ctx.Variables = saved
	}()

	return e.Eval(condition)
}

// buildEnv creates the evaluation environment
func (e *Evaluator) buildEnv() map[string]interface{} {
	env := map[string]interface{}{
		// Hardware
		"arch":               e.ctx.Hardware.Arch,
		"cpu_cores":          e.ctx.Hardware.CPUCores,
		"cpu_threads":        e.ctx.Hardware.CPUThreads,
		"cpu_model":          e.ctx.Hardware.CPUModel,
		"ram_mb":             e.ctx.Hardware.RAMMB,
		"ram_gb":             e.ctx.Hardware.RAMGB,
		"disk_gb":            e.ctx.Hardware.DiskGB,
		"has_gpu":            e.ctx.Hardware.HasGPU,
		"has_nvidia":         e.ctx.Hardware.HasNVIDIA,
		"has_amd":            e.ctx.Hardware.HasAMD,
		"has_intel":          e.ctx.Hardware.HasIntel,
		"has_virtualization": e.ctx.Hardware.HasVirtualization,

		// Environment
		"user":         e.ctx.Environment.User,
		"is_root":      e.ctx.Environment.IsRoot,
		"has_display":  e.ctx.Environment.HasDisplay,
		"display_type": e.ctx.Environment.DisplayType,
		"desktop_env":  e.ctx.Environment.DesktopEnv,
		"hostname":     e.ctx.Environment.Hostname,
		"home_dir":     e.ctx.Environment.HomeDir,

		// Variables
		"vars": e.ctx.Variables,
	}

	// Add variables as top-level keys for convenience
	for k, v := range e.ctx.Variables {
		if _, exists := env[k]; !exists {
			env[k] = v
		}
	}

	return env
}

// DetectHardware detects hardware and returns HardwareEnv
func DetectHardware() (*HardwareEnv, error) {
	hw := &HardwareEnv{
		Arch: runtime.GOARCH,
	}

	// Detect CPU cores
	if out, err := exec.Command("nproc").Output(); err == nil {
		if n, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil {
			hw.CPUThreads = n
			hw.CPUCores = n / 2 // Approximate
			if hw.CPUCores < 1 {
				hw.CPUCores = n
			}
		}
	}

	// Detect RAM
	if out, err := exec.Command("free", "-m").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Mem:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if mb, err := strconv.Atoi(fields[1]); err == nil {
						hw.RAMMB = mb
						hw.RAMGB = mb / 1024
					}
				}
				break
			}
		}
	}

	// Detect GPU
	if out, err := exec.Command("lspci").Output(); err == nil {
		outStr := strings.ToLower(string(out))
		hw.HasNVIDIA = strings.Contains(outStr, "nvidia")
		hw.HasAMD = strings.Contains(outStr, "amd") || strings.Contains(outStr, "radeon")
		hw.HasIntel = strings.Contains(outStr, "intel") && (strings.Contains(outStr, "vga") || strings.Contains(outStr, "display"))
		hw.HasGPU = hw.HasNVIDIA || hw.HasAMD || hw.HasIntel
	}

	// Detect virtualization
	if out, err := exec.Command("lscpu").Output(); err == nil {
		outStr := strings.ToLower(string(out))
		hw.HasVirtualization = strings.Contains(outStr, "vmx") || strings.Contains(outStr, "svm")
	}

	return hw, nil
}

// DetectEnvironment detects system environment
func DetectEnvironment() (*Environment, error) {
	env := &Environment{}

	// User
	if user := os.Getenv("USER"); user != "" {
		env.User = user
	}
	env.IsRoot = os.Geteuid() == 0

	// Hostname
	if hostname, err := os.Hostname(); err == nil {
		env.Hostname = hostname
	}

	// Home dir
	if home := os.Getenv("HOME"); home != "" {
		env.HomeDir = home
	}

	// Display
	env.HasDisplay = os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		env.DisplayType = "wayland"
	} else if os.Getenv("DISPLAY") != "" {
		env.DisplayType = "x11"
	} else {
		env.DisplayType = "none"
	}

	// Desktop environment
	env.DesktopEnv = os.Getenv("XDG_CURRENT_DESKTOP")
	if env.DesktopEnv == "" {
		env.DesktopEnv = os.Getenv("DESKTOP_SESSION")
	}

	return env, nil
}

// ValidateExpression validates a condition expression
func ValidateExpression(condition string) error {
	_, err := expr.Compile(condition, expr.Env(map[string]interface{}{
		"arch":               "",
		"cpu_cores":          0,
		"cpu_threads":        0,
		"cpu_model":          "",
		"ram_mb":             0,
		"ram_gb":             0,
		"disk_gb":            0,
		"has_gpu":            false,
		"has_nvidia":         false,
		"has_amd":            false,
		"has_intel":          false,
		"has_virtualization": false,
		"user":               "",
		"is_root":            false,
		"has_display":        false,
		"display_type":       "",
		"desktop_env":        "",
		"hostname":           "",
		"home_dir":           "",
		"vars":               map[string]interface{}{},
	}), expr.AsBool())
	return err
}

// ExtractVariables extracts variable references from a condition
func ExtractVariables(condition string) []string {
	var vars []string
	seen := make(map[string]bool)

	// Simple tokenization - split by operators and keywords
	// Remove string literals first
	cleaned := removeStrings(condition)

	words := strings.FieldsFunc(cleaned, func(r rune) bool {
		return !isIdentChar(r)
	})

	for _, word := range words {
		// Skip numbers, operators, keywords, strings
		if _, err := strconv.ParseFloat(word, 64); err == nil {
			continue
		}
		if isKeyword(word) {
			continue
		}
		if len(word) < 2 { // Skip single-char tokens
			continue
		}
		if !isBuiltinVariable(word) && !seen[word] {
			vars = append(vars, word)
			seen[word] = true
		}
	}

	return vars
}

func removeStrings(s string) string {
	var result strings.Builder
	inString := false
	escape := false

	for _, r := range s {
		if escape {
			escape = false
			continue
		}
		if r == '\\' {
			escape = true
			continue
		}
		if r == '"' || r == '\'' {
			inString = !inString
			continue
		}
		if !inString {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func isIdentChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '_' || r == '.'
}

func isKeyword(word string) bool {
	keywords := map[string]bool{
		"true": true, "false": true, "nil": true, "null": true,
		"and": true, "or": true, "not": true, "in": true,
		"contains": true, "matches": true, "startsWith": true, "endsWith": true,
		"len": true, "all": true, "none": true, "any": true, "one": true,
		"filter": true, "map": true, "count": true, "sum": true,
	}
	return keywords[word]
}

func isBuiltinVariable(name string) bool {
	builtins := map[string]bool{
		"arch": true, "cpu_cores": true, "cpu_threads": true, "cpu_model": true,
		"ram_mb": true, "ram_gb": true, "disk_gb": true,
		"has_gpu": true, "has_nvidia": true, "has_amd": true, "has_intel": true,
		"has_virtualization": true,
		"user":               true, "is_root": true, "has_display": true, "display_type": true,
		"desktop_env": true, "hostname": true, "home_dir": true,
		"vars": true,
	}
	return builtins[name]
}

// FormatError formats a condition error for display
func FormatError(err error, condition string) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("Condition error: %s\n  Expression: %s", err, condition)
}
