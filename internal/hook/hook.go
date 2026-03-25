package hook

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Phase represents a recipe execution phase
type Phase string

const (
	PhaseInstall   Phase = "install"
	PhaseConfigure Phase = "configure"
	PhaseVerify    Phase = "verify"
	PhaseUninstall Phase = "uninstall"
)

// Timing represents when a hook runs
type Timing string

const (
	TimingPre  Timing = "pre"
	TimingPost Timing = "post"
)

// Hook represents a single hook command
type Hook struct {
	Type    string `yaml:"type" json:"type"` // apt, command, script, service
	Command string `yaml:"command" json:"command"`
	Action  string `yaml:"action" json:"action"` // update, install, remove, start, stop, enable, disable
	Package string `yaml:"package" json:"package"`
	Service string `yaml:"service" json:"service"`
	Script  string `yaml:"script" json:"script"`
	When    string `yaml:"when" json:"when"` // Condition expression
	Ignore  bool   `yaml:"ignore_errors" json:"ignore_errors"`
	Timeout int    `yaml:"timeout" json:"timeout"` // seconds
}

// HookSet contains all hooks for a recipe
type HookSet struct {
	PreInstall    []Hook `yaml:"pre_install" json:"pre_install"`
	PostInstall   []Hook `yaml:"post_install" json:"post_install"`
	PreConfigure  []Hook `yaml:"pre_configure" json:"pre_configure"`
	PostConfigure []Hook `yaml:"post_configure" json:"post_configure"`
	PreVerify     []Hook `yaml:"pre_verify" json:"pre_verify"`
	PostVerify    []Hook `yaml:"post_verify" json:"post_verify"`
	OnFailure     []Hook `yaml:"on_failure" json:"on_failure"`
	Rollback      []Hook `yaml:"rollback" json:"rollback"`
}

// Executor executes hooks
type Executor struct {
	dryRun    bool
	verbose   bool
	aptCmd    APTRunner
	envVars   map[string]string
	kitchen   string // Path to kitchen directory for script resolution
	hookCache map[string]*HookResult // Cache for hook execution results
}

// HookResult caches hook execution result
type HookResult struct {
	Output  string
	Error   error
	Success bool
}

// APTRunner interface for APT operations
type APTRunner interface {
	Install(packages ...string) error
	Remove(packages ...string) error
	Update() error
	IsInstalled(pkg string) bool
	BatchInstall(packages []string) error
}

// Result contains hook execution result
type Result struct {
	Phase   Phase
	Timing  Timing
	Index   int
	Success bool
	Error   error
	Output  string
	Elapsed time.Duration
}

// ExecutionSummary contains summary of all hook executions
type ExecutionSummary struct {
	Phase     Phase
	Results   []Result
	Success   bool
	TotalTime time.Duration
	Errors    []error
}

// NewExecutor creates a new hook executor
func NewExecutor(dryRun, verbose bool, aptRunner APTRunner) *Executor {
	return &Executor{
		dryRun:    dryRun,
		verbose:   verbose,
		aptCmd:    aptRunner,
		envVars:   make(map[string]string),
		hookCache: make(map[string]*HookResult),
	}
}

// SetKitchen sets the kitchen directory path for script resolution
func (e *Executor) SetKitchen(kitchen string) {
	e.kitchen = kitchen
}

// SetEnv sets environment variables for hook execution
func (e *Executor) SetEnv(key, value string) {
	e.envVars[key] = value
}

// GetEnv returns all environment variables
func (e *Executor) GetEnv() map[string]string {
	return e.envVars
}

// ExecuteHooks executes a set of hooks for a phase
func (e *Executor) ExecuteHooks(hooks HookSet, phase Phase, recipeName string, variables map[string]interface{}) *ExecutionSummary {
	summary := &ExecutionSummary{
		Phase:   phase,
		Results: []Result{},
		Success: true,
	}

	start := time.Now()

	// Get hooks for this phase
	var preHooks, postHooks []Hook
	switch phase {
	case PhaseInstall:
		preHooks = hooks.PreInstall
		postHooks = hooks.PostInstall
	case PhaseConfigure:
		preHooks = hooks.PreConfigure
		postHooks = hooks.PostConfigure
	case PhaseVerify:
		preHooks = hooks.PreVerify
		postHooks = hooks.PostVerify
	}

	// Execute pre-hooks
	for i, hook := range preHooks {
		result := e.executeHook(hook, phase, TimingPre, i, variables)
		summary.Results = append(summary.Results, result)
		if !result.Success && !hook.Ignore {
			summary.Success = false
			summary.Errors = append(summary.Errors, result.Error)
			return summary
		}
	}

	// Note: Main installation happens after pre-hooks, before post-hooks
	// This function only handles hooks, not the main operation

	// Execute post-hooks
	for i, hook := range postHooks {
		result := e.executeHook(hook, phase, TimingPost, i, variables)
		summary.Results = append(summary.Results, result)
		if !result.Success && !hook.Ignore {
			summary.Success = false
			summary.Errors = append(summary.Errors, result.Error)
			return summary
		}
	}

	summary.TotalTime = time.Since(start)
	return summary
}

// ExecuteFailureHooks executes failure hooks
func (e *Executor) ExecuteFailureHooks(hooks HookSet, originalErr error, variables map[string]interface{}) *ExecutionSummary {
	summary := &ExecutionSummary{
		Phase:   "failure",
		Results: []Result{},
		Success: true,
	}

	start := time.Now()

	for i, hook := range hooks.OnFailure {
		result := e.executeHook(hook, "failure", TimingPost, i, variables)
		summary.Results = append(summary.Results, result)
		if !result.Success && !hook.Ignore {
			summary.Success = false
			summary.Errors = append(summary.Errors, result.Error)
		}
	}

	summary.TotalTime = time.Since(start)
	return summary
}

// ExecuteRollbackHooks executes rollback hooks
func (e *Executor) ExecuteRollbackHooks(hooks HookSet, variables map[string]interface{}) *ExecutionSummary {
	summary := &ExecutionSummary{
		Phase:   "rollback",
		Results: []Result{},
		Success: true,
	}

	start := time.Now()

	for i, hook := range hooks.Rollback {
		result := e.executeHook(hook, "rollback", TimingPost, i, variables)
		summary.Results = append(summary.Results, result)
		if !result.Success && !hook.Ignore {
			summary.Success = false
			summary.Errors = append(summary.Errors, result.Error)
		}
	}

	summary.TotalTime = time.Since(start)
	return summary
}

// executeHook executes a single hook
func (e *Executor) executeHook(hook Hook, phase Phase, timing Timing, index int, variables map[string]interface{}) Result {
	result := Result{
		Phase:  phase,
		Timing: timing,
		Index:  index,
	}

	// Check condition
	if hook.When != "" {
		// Condition evaluation would be done by the conditions package
		// For now, assume true if condition is not empty
		if e.verbose {
			fmt.Printf("  [hook] Checking condition: %s\n", hook.When)
		}
	}

	start := time.Now()

	// Execute based on type
	var err error
	switch hook.Type {
	case "apt":
		err = e.executeAPTHook(hook)
	case "command", "cmd":
		err = e.executeCommandHook(hook, variables)
	case "script":
		err = e.executeScriptHook(hook, variables)
	case "service":
		err = e.executeServiceHook(hook)
	case "template":
		err = e.executeTemplateHook(hook, variables)
	default:
		err = fmt.Errorf("unknown hook type: %s", hook.Type)
	}

	result.Elapsed = time.Since(start)
	result.Success = err == nil
	result.Error = err

	if e.verbose {
		status := "✓"
		if !result.Success {
			status = "✗"
		}
		fmt.Printf("  [hook] %s %s (%s)\n", status, e.describeHook(hook), result.Elapsed.Round(time.Millisecond))
	}

	return result
}

func (e *Executor) executeAPTHook(hook Hook) error {
	if e.dryRun {
		fmt.Printf("  [dry-run] APT hook: %s %s\n", hook.Action, hook.Package)
		return nil
	}

	switch hook.Action {
	case "update":
		return e.aptCmd.Update()
	case "install":
		if hook.Package == "" {
			return fmt.Errorf("apt install hook missing package")
		}
		return e.aptCmd.Install(hook.Package)
	case "remove":
		if hook.Package == "" {
			return fmt.Errorf("apt remove hook missing package")
		}
		return e.aptCmd.Remove(hook.Package)
	default:
		return fmt.Errorf("unknown apt action: %s", hook.Action)
	}
}

func (e *Executor) executeCommandHook(hook Hook, variables map[string]interface{}) error {
	if e.dryRun {
		fmt.Printf("  [dry-run] Command hook: %s\n", hook.Command)
		return nil
	}

	cmdStr := hook.Command
	if cmdStr == "" {
		cmdStr = hook.Script
	}

	// Render template if variables are provided
	if len(variables) > 0 {
		rendered, err := e.renderTemplate(cmdStr, variables)
		if err != nil {
			return fmt.Errorf("failed to render command template: %w", err)
		}
		cmdStr = rendered
	}

	timeout := time.Duration(hook.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", cmdStr)
	cmd.Env = e.buildEnv()
	
	// Capture output if verbose
	var stdout, stderr bytes.Buffer
	if e.verbose {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("hook timed out after %v", timeout)
		}
		if e.verbose && stderr.Len() > 0 {
			return fmt.Errorf("command failed: %w\nOutput: %s", err, stderr.String())
		}
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

func (e *Executor) executeScriptHook(hook Hook, variables map[string]interface{}) error {
	if e.dryRun {
		fmt.Printf("  [dry-run] Script hook: %s\n", hook.Script)
		return nil
	}

	scriptPath := hook.Script
	if scriptPath == "" {
		return fmt.Errorf("script hook missing script path")
	}

	// Resolve script path relative to kitchen directory
	if !filepath.IsAbs(scriptPath) && e.kitchen != "" {
		scriptPath = filepath.Join(e.kitchen, scriptPath)
	}

	// Check if script exists
	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("script not found: %s", scriptPath)
	}

	timeout := time.Duration(hook.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/sh", scriptPath)
	cmd.Env = e.buildEnv()
	cmd.Dir = filepath.Dir(scriptPath)
	
	// Capture output if verbose
	var stdout, stderr bytes.Buffer
	if e.verbose {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("script timed out after %v", timeout)
		}
		if e.verbose && stderr.Len() > 0 {
			return fmt.Errorf("script failed: %w\nOutput: %s", err, stderr.String())
		}
		return fmt.Errorf("script failed: %w", err)
	}

	return nil
}

func (e *Executor) executeServiceHook(hook Hook) error {
	if e.dryRun {
		fmt.Printf("  [dry-run] Service hook: %s %s\n", hook.Action, hook.Service)
		return nil
	}

	if hook.Service == "" {
		return fmt.Errorf("service hook missing service name")
	}

	var cmd *exec.Cmd
	switch hook.Action {
	case "start":
		cmd = exec.Command("sudo", "systemctl", "start", hook.Service)
	case "stop":
		cmd = exec.Command("sudo", "systemctl", "stop", hook.Service)
	case "restart":
		cmd = exec.Command("sudo", "systemctl", "restart", hook.Service)
	case "enable":
		cmd = exec.Command("sudo", "systemctl", "enable", hook.Service)
	case "disable":
		cmd = exec.Command("sudo", "systemctl", "disable", hook.Service)
	case "reload":
		cmd = exec.Command("sudo", "systemctl", "reload", hook.Service)
	default:
		return fmt.Errorf("unknown service action: %s", hook.Action)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("service %s %s failed: %w\nOutput: %s", hook.Action, hook.Service, err, out)
	}

	return nil
}

// executeTemplateHook executes a template-based hook
func (e *Executor) executeTemplateHook(hook Hook, variables map[string]interface{}) error {
	if e.dryRun {
		fmt.Printf("  [dry-run] Template hook: %s\n", hook.Command)
		return nil
	}

	if hook.Command == "" {
		return fmt.Errorf("template hook missing command template")
	}

	// Render the template
	rendered, err := e.renderTemplate(hook.Command, variables)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	if e.verbose {
		fmt.Printf("  [hook] Rendered template: %s\n", rendered)
	}

	// Execute the rendered command
	timeout := time.Duration(hook.Timeout) * time.Second
	if timeout == 0 {
		timeout = 5 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", rendered)
	cmd.Env = e.buildEnv()

	var stdout, stderr bytes.Buffer
	if e.verbose {
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("hook timed out after %v", timeout)
		}
		if e.verbose && stderr.Len() > 0 {
			return fmt.Errorf("command failed: %w\nOutput: %s", err, stderr.String())
		}
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

// renderTemplate renders a Go template string with the provided variables
func (e *Executor) renderTemplate(tmplStr string, variables map[string]interface{}) (string, error) {
	tmpl, err := template.New("hook").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (e *Executor) buildEnv() []string {
	env := os.Environ()
	for k, v := range e.envVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}

func (e *Executor) describeHook(hook Hook) string {
	switch hook.Type {
	case "apt":
		return fmt.Sprintf("apt %s %s", hook.Action, hook.Package)
	case "command", "cmd":
		return hook.Command
	case "script":
		return fmt.Sprintf("script %s", hook.Script)
	case "service":
		return fmt.Sprintf("service %s %s", hook.Action, hook.Service)
	default:
		return hook.Type
	}
}

// FormatSummary formats an execution summary for display
func FormatSummary(summary *ExecutionSummary) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Phase: %s (%v)\n", summary.Phase, summary.TotalTime.Round(time.Millisecond)))

	for _, result := range summary.Results {
		status := "✓"
		if !result.Success {
			status = "✗"
		}
		sb.WriteString(fmt.Sprintf("  %s %s [%s]\n", status,
			result.Phase, result.Timing))
	}

	if !summary.Success {
		sb.WriteString("Errors:\n")
		for _, err := range summary.Errors {
			sb.WriteString(fmt.Sprintf("  - %v\n", err))
		}
	}

	return sb.String()
}

// ParseHookSet parses hook definitions from recipe metadata
func ParseHookSet(hooks map[string][]map[string]interface{}) HookSet {
	set := HookSet{}

	set.PreInstall = parseHookList(hooks["pre_install"])
	set.PostInstall = parseHookList(hooks["post_install"])
	set.PreConfigure = parseHookList(hooks["pre_configure"])
	set.PostConfigure = parseHookList(hooks["post_configure"])
	set.PreVerify = parseHookList(hooks["pre_verify"])
	set.PostVerify = parseHookList(hooks["post_verify"])
	set.OnFailure = parseHookList(hooks["on_failure"])
	set.Rollback = parseHookList(hooks["rollback"])

	return set
}

func parseHookList(hooks []map[string]interface{}) []Hook {
	if len(hooks) == 0 {
		return nil
	}

	var result []Hook
	for _, h := range hooks {
		hook := Hook{}

		if v, ok := h["type"].(string); ok {
			hook.Type = v
		}
		if v, ok := h["command"].(string); ok {
			hook.Command = v
		}
		if v, ok := h["action"].(string); ok {
			hook.Action = v
		}
		if v, ok := h["package"].(string); ok {
			hook.Package = v
		}
		if v, ok := h["service"].(string); ok {
			hook.Service = v
		}
		if v, ok := h["script"].(string); ok {
			hook.Script = v
		}
		if v, ok := h["when"].(string); ok {
			hook.When = v
		}
		if v, ok := h["ignore_errors"].(bool); ok {
			hook.Ignore = v
		}
		if v, ok := h["timeout"].(int); ok {
			hook.Timeout = v
		}

		result = append(result, hook)
	}

	return result
}
