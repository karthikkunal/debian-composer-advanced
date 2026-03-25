package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Check struct {
	Name        string
	Description string
	Blocking    bool
	Run         func(projectRoot string, stagedFiles []string) (bool, string)
	SkipIf      func(stagedFiles []string) bool
	EnvFlag     string // Environment variable to enable/disable
}

var (
	debug   = os.Getenv("DEBUG") == "true"
	verbose = os.Getenv("VERBOSE") == "true"
	noColor = os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"

	// Colors (disabled if NO_COLOR set or not a terminal)
	green  = "\033[0;32m"
	red    = "\033[0;31m"
	yellow = "\033[1;33m"
	blue   = "\033[0;34m"
	cyan   = "\033[0;36m"
	nc     = "\033[0m"

	// Icons
	iconPass = "✓"
	iconFail = "✗"
	iconWarn = "⚠"
	iconSkip = "⊘"
	iconInfo = "ℹ"
)

func init() {
	if noColor {
		green = ""
		red = ""
		yellow = ""
		blue = ""
		cyan = ""
		nc = ""
	}
}

func main() {
	projectRoot, err := getProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s%s Error: Could not determine project root%s\n", red, iconFail, nc)
		os.Exit(1)
	}

	stagedFiles := getStagedFiles(projectRoot)
	changedOnly := len(stagedFiles) > 0

	if verbose {
		debug = true
	}

	printHeader("Pre-commit Checks (Go)")
	fmt.Printf(" Project: %s\n", projectRoot)
	if changedOnly {
		fmt.Printf(" Mode: changed-only (%d staged files)\n", len(stagedFiles))
	} else {
		fmt.Println(" Mode: full scan")
	}
	if debug {
		fmt.Println(" Debug: enabled")
	}
	fmt.Println()

	checks := []Check{
		{
			Name:        "gofmt",
			Description: "Go code formatting",
			Blocking:    true,
			Run:         checkGoFmt,
			SkipIf: func(staged []string) bool {
				return !hasFilesMatching(staged, ".go")
			},
			EnvFlag: "ENABLE_GOFMT_CHECK",
		},
		{
			Name:        "gobuild",
			Description: "Go compilation",
			Blocking:    true,
			Run:         checkGoBuild,
			SkipIf: func(staged []string) bool {
				return !hasFilesMatching(staged, ".go")
			},
			EnvFlag: "ENABLE_GOBUILD_CHECK",
		},
		{
			Name:        "govet",
			Description: "Go static analysis",
			Blocking:    false,
			Run:         checkGoVet,
			SkipIf: func(staged []string) bool {
				return !hasFilesMatching(staged, ".go")
			},
			EnvFlag: "ENABLE_GOVET_CHECK",
		},
		{
			Name:        "permissions",
			Description: "Executable permissions",
			Blocking:    false,
			Run:         checkPermissions,
			EnvFlag:     "ENABLE_PERMISSIONS_CHECK",
		},
		{
			Name:        "yamllint",
			Description: "YAML linting",
			Blocking:    true,
			Run:         checkYamlLint,
			SkipIf: func(staged []string) bool {
				return !hasFilesMatching(staged, ".yaml") && !hasFilesMatching(staged, ".yml")
			},
			EnvFlag: "ENABLE_YAMLLINT_CHECK",
		},
	}

	failed := 0
	checksRun := 0
	checksPassed := 0
	checksFailed := 0
	checksSkipped := 0

	for _, check := range checks {
		// Check if check is disabled via environment
		if check.EnvFlag != "" && os.Getenv(check.EnvFlag) == "false" {
			fmt.Printf("Running: %s - %s\n", check.Name, check.Description)
			fmt.Printf("  %s%s disabled via %s%s\n", cyan, iconInfo, check.EnvFlag, nc)
			checksSkipped++
			checksRun++
			if verbose {
				fmt.Println()
			}
			continue
		}

		// Skip check if no relevant files staged
		if changedOnly && check.SkipIf != nil && check.SkipIf(stagedFiles) {
			fmt.Printf("Running: %s - %s\n", check.Name, check.Description)
			fmt.Printf("  %s%s skipped (no relevant files)%s\n", yellow, iconSkip, nc)
			checksSkipped++
			checksRun++
			if verbose {
				fmt.Println()
			}
			continue
		}

		start := time.Now()
		fmt.Printf("Running: %s - %s\n", check.Name, check.Description)

		passed, msg := check.Run(projectRoot, stagedFiles)
		duration := time.Since(start)

		if passed {
			fmt.Printf("  %s%s OK%s", green, iconPass, nc)
			if msg != "" {
				fmt.Printf(" (%s)", msg)
			}
			fmt.Printf(" [%s]\n", duration.Round(time.Millisecond))
			checksPassed++
		} else {
			fmt.Printf("  %s%s FAILED%s", red, iconFail, nc)
			if msg != "" {
				fmt.Printf(": %s", msg)
			}
			fmt.Printf(" [%s]\n", duration.Round(time.Millisecond))
			checksFailed++
			if check.Blocking {
				failed++
			}
		}
		checksRun++
		if verbose {
			fmt.Println()
		}
	}

	printSummary("Pre-commit", checksRun, checksPassed, checksFailed, checksSkipped)

	if failed == 0 {
		fmt.Printf("%s%s Pre-commit checks passed!%s\n", green, iconPass, nc)
	} else {
		fmt.Printf("%s%s Pre-commit checks failed!%s\n", red, iconFail, nc)
		fmt.Println()
		fmt.Println("To bypass pre-commit hooks (not recommended):")
		fmt.Println("  git commit --no-verify")
		os.Exit(1)
	}
}

func printHeader(title string) {
	fmt.Println("============================================================================")
	fmt.Printf(" %s\n", title)
	fmt.Println("============================================================================")
}

func printSummary(hookType string, total, passed, failed, skipped int) {
	fmt.Println()
	fmt.Println("============================================================================")
	fmt.Printf(" %s Check Summary\n", hookType)
	fmt.Println("============================================================================")
	fmt.Printf(" Total checks: %d\n", total)
	if passed > 0 {
		fmt.Printf(" %s%s Passed: %d%s\n", green, iconPass, passed, nc)
	} else {
		fmt.Printf("   Passed: 0\n")
	}
	if failed > 0 {
		fmt.Printf(" %s%s Failed: %d%s\n", red, iconFail, failed, nc)
	} else {
		fmt.Printf(" %s%s Failed: 0%s\n", green, iconPass, nc)
	}
	if skipped > 0 {
		fmt.Printf(" %s%s Skipped: %d%s\n", yellow, iconSkip, skipped, nc)
	}
	fmt.Println("============================================================================")
	fmt.Println()
}

func getProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return os.Getwd()
}

func getStagedFiles(projectRoot string) []string {
	cmd := exec.Command("git", "diff", "--cached", "--name-only", "--diff-filter=ACM")
	cmd.Dir = projectRoot
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, f := range files {
		if f != "" {
			result = append(result, f)
		}
	}
	return result
}

func hasFilesMatching(files []string, ext string) bool {
	for _, f := range files {
		if strings.HasSuffix(f, ext) {
			return true
		}
	}
	return false
}

func runCmd(projectRoot, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func checkGoFmt(projectRoot string, stagedFiles []string) (bool, string) {
	output, err := runCmd(projectRoot, "gofmt", "-l", ".")
	if err != nil {
		return false, fmt.Sprintf("gofmt failed: %v", err)
	}

	files := strings.Split(strings.TrimSpace(output), "\n")
	var unformatted []string
	for _, f := range files {
		if f != "" && !strings.Contains(f, "vendor/") {
			if len(stagedFiles) > 0 && !isFileStaged(f, stagedFiles) {
				continue
			}
			unformatted = append(unformatted, f)
		}
	}

	if len(unformatted) == 0 {
		return true, ""
	}

	msg := fmt.Sprintf("%d files need formatting", len(unformatted))
	if len(unformatted) <= 5 {
		msg += ": " + strings.Join(unformatted, ", ")
	} else {
		msg += ": " + strings.Join(unformatted[:5], ", ") + fmt.Sprintf(" ... and %d more", len(unformatted)-5)
	}
	return false, msg
}

func isFileStaged(file string, stagedFiles []string) bool {
	for _, staged := range stagedFiles {
		if staged == file || strings.HasSuffix(file, staged) {
			return true
		}
	}
	return false
}

func checkGoBuild(projectRoot string, stagedFiles []string) (bool, string) {
	if debug {
		fmt.Printf("    %s Running: go build -o /dev/null ./cmd/...\n", iconInfo)
	}
	_, err := runCmd(projectRoot, "go", "build", "-o", "/dev/null", "./cmd/...")
	if err != nil {
		return false, err.Error()
	}
	return true, ""
}

func checkGoVet(projectRoot string, stagedFiles []string) (bool, string) {
	_, err := runCmd(projectRoot, "go", "vet", "./...")
	if err != nil {
		return false, "warnings found"
	}
	return true, ""
}

func checkPermissions(projectRoot string, stagedFiles []string) (bool, string) {
	issues := 0
	var details []string

	hookBinaries := []string{
		"hooks/cmd/pre-commit/pre-commit",
		"hooks/cmd/pre-push/pre-push",
	}
	for _, f := range hookBinaries {
		path := filepath.Join(projectRoot, f)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.Mode()&0111 == 0 {
			issues++
			details = append(details, f+" not executable")
		}
	}

	binPath := filepath.Join(projectRoot, "bin", "debian-composer-go")
	if info, err := os.Stat(binPath); err == nil {
		if info.Mode()&0111 == 0 {
			issues++
			details = append(details, "bin/debian-composer-go not executable")
		}
	}

	if issues == 0 {
		return true, ""
	}
	return false, fmt.Sprintf("%d issues: %s", issues, strings.Join(details, "; "))
}

func checkYamlLint(projectRoot string, stagedFiles []string) (bool, string) {
	yamllintBin := os.Getenv("YAMLLINT_BIN")
	if yamllintBin == "" {
		yamllintBin = "yamllint"
	}

	if _, err := exec.LookPath(yamllintBin); err != nil {
		return true, "yamllint not installed (skipped)"
	}

	var yamlFiles []string

	if len(stagedFiles) > 0 {
		for _, f := range stagedFiles {
			if strings.HasSuffix(f, ".yaml") || strings.HasSuffix(f, ".yml") {
				yamlFiles = append(yamlFiles, f)
			}
		}
	} else {
		filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				base := filepath.Base(path)
				if strings.HasPrefix(base, ".") || base == "vendor" || base == "test_helper" {
					return filepath.SkipDir
				}
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".yaml" || ext == ".yml" {
				relPath, _ := filepath.Rel(projectRoot, path)
				yamlFiles = append(yamlFiles, relPath)
			}
			return nil
		})
	}

	if len(yamlFiles) == 0 {
		return true, "no YAML files"
	}

	if debug {
		fmt.Printf("    %s Checking %d YAML files\n", iconInfo, len(yamlFiles))
	}

	configFile := filepath.Join(projectRoot, ".yamllint.yml")
	args := []string{"-f", "parsable"}
	if _, err := os.Stat(configFile); err == nil {
		args = append(args, "-c", configFile)
	}
	args = append(args, yamlFiles...)

	output, err := runCmd(projectRoot, yamllintBin, args...)
	if err != nil {
		lines := strings.Split(output, "\n")
		errorCount := 0
		for _, line := range lines {
			if strings.Contains(line, "[error]") {
				errorCount++
			}
		}
		if errorCount > 0 {
			return false, fmt.Sprintf("%d errors", errorCount)
		}
		return true, "warnings only"
	}

	return true, fmt.Sprintf("%d files", len(yamlFiles))
}
