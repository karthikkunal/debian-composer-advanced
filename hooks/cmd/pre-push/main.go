package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Check struct {
	Name      string
	Run       func(projectRoot string) (CheckResult, error)
	Fatal     bool
	Condition func() bool
	EnvFlag   string
}

type CheckResult struct {
	Status  string
	Message string
}

var (
	debug   = os.Getenv("DEBUG") == "true"
	verbose = os.Getenv("VERBOSE") == "true"
	noColor = os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"

	green  = "\033[0;32m"
	red    = "\033[0;31m"
	yellow = "\033[1;33m"
	blue   = "\033[0;34m"
	cyan   = "\033[0;36m"
	nc     = "\033[0m"

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

	if verbose {
		debug = true
	}

	fmt.Printf("%s%s Running pre-push checks (Go)...%s\n", blue, iconInfo, nc)

	start := time.Now()
	failed := 0
	checkNum := 1
	totalChecks := 10

	checks := []Check{
		{Name: "Go build", Run: checkGoBuild, Fatal: true, EnvFlag: "ENABLE_GOBUILD_CHECK"},
		{Name: "Go test (unit)", Run: checkGoTest, EnvFlag: "ENABLE_GOTEST_CHECK"},
		{Name: "Go vet", Run: checkGoVet, EnvFlag: "ENABLE_GOVET_CHECK"},
		{Name: "Go fmt", Run: checkGoFmt, EnvFlag: "ENABLE_GOFMT_CHECK"},
		{Name: "StaticCheck", Run: checkStaticCheck, Condition: func() bool {
			return commandExists("staticcheck")
		}, EnvFlag: "ENABLE_STATICCHECK_CHECK"},
		{Name: "Binary smoke test", Run: checkBinarySmoke, Fatal: true, EnvFlag: "ENABLE_SMOKE_CHECK"},
		{Name: "Test coverage", Run: checkCoverage, EnvFlag: "ENABLE_COVERAGE_CHECK"},
		{Name: "YAML lint", Run: checkYamlLint, Condition: func() bool {
			return commandExists(getEnvOrDefault("YAMLLINT_BIN", "yamllint"))
		}, EnvFlag: "ENABLE_YAMLLINT_CHECK"},
		{Name: "Blends validation", Run: checkBlends, Condition: func() bool {
			return commandExists("yq")
		}, EnvFlag: "ENABLE_BLENDS_CHECK"},
		{Name: "Documentation", Run: checkDocumentation, EnvFlag: "ENABLE_DOCUMENTATION_CHECK"},
	}

	for _, check := range checks {
		fmt.Printf("  [%d/%d] %s... ", checkNum, totalChecks, check.Name)

		// Check if disabled via environment
		if check.EnvFlag != "" && os.Getenv(check.EnvFlag) == "false" {
			fmt.Printf("%s%s disabled%s\n", cyan, iconSkip, nc)
			checkNum++
			continue
		}

		if check.Condition != nil && !check.Condition() {
			fmt.Printf("%s%s skipped (not found)%s\n", yellow, iconSkip, nc)
			checkNum++
			continue
		}

		checkStart := time.Now()
		result, err := check.Run(projectRoot)
		duration := time.Since(checkStart)

		if err != nil {
			fmt.Printf("%s%s ERROR%s: %v [%s]\n", red, iconFail, nc, err, duration.Round(time.Millisecond))
			if check.Fatal {
				failed = 1
			}
			checkNum++
			continue
		}

		switch result.Status {
		case "ok":
			msg := fmt.Sprintf("%s%s OK%s", green, iconPass, nc)
			if result.Message != "" {
				msg = fmt.Sprintf("%s%s OK (%s)%s", green, iconPass, result.Message, nc)
			}
			fmt.Printf("%s [%s]\n", msg, duration.Round(time.Millisecond))
		case "failed":
			fmt.Printf("%s%s FAILED%s", red, iconFail, nc)
			if result.Message != "" {
				fmt.Printf(": %s", result.Message)
			}
			fmt.Printf(" [%s]\n", duration.Round(time.Millisecond))
			if check.Fatal {
				failed = 1
			}
		case "warning":
			fmt.Printf("%s%s Warnings%s", yellow, iconWarn, nc)
			if result.Message != "" {
				fmt.Printf(": %s", result.Message)
			}
			fmt.Printf(" [%s]\n", duration.Round(time.Millisecond))
		default:
			fmt.Printf("%s%s %s%s\n", yellow, iconWarn, result.Status, nc)
		}

		checkNum++
	}

	totalDuration := time.Since(start)
	fmt.Println()
	if failed == 0 {
		fmt.Printf("%s%s Pre-push checks passed!%s (total: %s)\n", green, iconPass, nc, totalDuration.Round(time.Millisecond))
	} else {
		fmt.Printf("%s%s Pre-push checks failed!%s (total: %s)\n", red, iconFail, nc, totalDuration.Round(time.Millisecond))
		os.Exit(1)
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
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

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func runCmd(projectRoot, name string, args ...string) (string, error) {
	if debug {
		fmt.Printf("\n    %s Running: %s %s\n", iconInfo, name, strings.Join(args, " "))
	}
	cmd := exec.Command(name, args...)
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func checkGoBuild(projectRoot string) (CheckResult, error) {
	_, err := runCmd(projectRoot, "go", "build", "-o", "/dev/null", "./cmd/...")
	if err != nil {
		return CheckResult{Status: "failed", Message: err.Error()}, nil
	}
	return CheckResult{Status: "ok"}, nil
}

func checkGoTest(projectRoot string) (CheckResult, error) {
	_, err := runCmd(projectRoot, "go", "test", "-short", "-count=1", "./...")
	if err != nil {
		return CheckResult{Status: "warning", Message: "tests failed"}, nil
	}
	return CheckResult{Status: "ok"}, nil
}

func checkGoVet(projectRoot string) (CheckResult, error) {
	_, err := runCmd(projectRoot, "go", "vet", "./...")
	if err != nil {
		return CheckResult{Status: "warning", Message: "vet warnings"}, nil
	}
	return CheckResult{Status: "ok"}, nil
}

func checkGoFmt(projectRoot string) (CheckResult, error) {
	output, err := runCmd(projectRoot, "gofmt", "-l", ".")
	if err != nil {
		return CheckResult{Status: "warning", Message: "gofmt failed"}, nil
	}

	files := strings.Split(strings.TrimSpace(output), "\n")
	var unformatted []string
	for _, f := range files {
		if f != "" && !strings.Contains(f, "vendor/") {
			unformatted = append(unformatted, f)
		}
	}

	if len(unformatted) == 0 {
		return CheckResult{Status: "ok"}, nil
	}
	return CheckResult{Status: "warning", Message: fmt.Sprintf("%d files need formatting", len(unformatted))}, nil
}

func checkStaticCheck(projectRoot string) (CheckResult, error) {
	_, err := runCmd(projectRoot, "staticcheck", "./...")
	if err != nil {
		return CheckResult{Status: "warning", Message: "static analysis warnings"}, nil
	}
	return CheckResult{Status: "ok"}, nil
}

func checkBinarySmoke(projectRoot string) (CheckResult, error) {
	binPath := filepath.Join(projectRoot, "bin", "debian-composer-go")

	if _, err := os.Stat(binPath); err != nil {
		_, err := runCmd(projectRoot, "go", "build", "-o", binPath, "./cmd/...")
		if err != nil {
			return CheckResult{Status: "failed", Message: "build failed"}, nil
		}
	}

	_, err := runCmd(projectRoot, binPath, "--help")
	if err != nil {
		return CheckResult{Status: "failed", Message: "help command failed"}, nil
	}
	return CheckResult{Status: "ok"}, nil
}

func checkCoverage(projectRoot string) (CheckResult, error) {
	minCoverage := getEnvOrDefault("MIN_COVERAGE", "70")
	minVal, _ := strconv.ParseFloat(minCoverage, 64)

	tmpFile, err := os.CreateTemp("", "coverage-*.out")
	if err != nil {
		return CheckResult{Status: "warning", Message: "cannot create temp file"}, nil
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	_, err = runCmd(projectRoot, "go", "test", "-coverprofile="+tmpFile.Name(), "./...")
	if err != nil {
		return CheckResult{Status: "warning", Message: "coverage test failed"}, nil
	}

	cmd := exec.Command("go", "tool", "cover", "-func="+tmpFile.Name())
	cmd.Dir = projectRoot
	output, err := cmd.Output()
	if err != nil {
		return CheckResult{Status: "warning", Message: "cannot parse coverage"}, nil
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "total:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				pctStr := strings.TrimSuffix(parts[2], "%")
				pct, err := strconv.ParseFloat(pctStr, 64)
				if err == nil {
					if pct >= minVal {
						return CheckResult{Status: "ok", Message: fmt.Sprintf("%.1f%% (min: %.0f%%)", pct, minVal)}, nil
					}
					return CheckResult{Status: "warning", Message: fmt.Sprintf("low coverage %.1f%% (min: %.0f%%)", pct, minVal)}, nil
				}
			}
		}
	}
	return CheckResult{Status: "warning", Message: "could not parse coverage"}, nil
}

func checkYamlLint(projectRoot string) (CheckResult, error) {
	yamllintBin := getEnvOrDefault("YAMLLINT_BIN", "yamllint")

	var yamlFiles []string
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

	if len(yamlFiles) == 0 {
		return CheckResult{Status: "ok", Message: "no YAML files"}, nil
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
			return CheckResult{Status: "failed", Message: fmt.Sprintf("%d errors", errorCount)}, nil
		}
		return CheckResult{Status: "ok", Message: "warnings only"}, nil
	}

	return CheckResult{Status: "ok", Message: fmt.Sprintf("%d files", len(yamlFiles))}, nil
}

func checkBlends(projectRoot string) (CheckResult, error) {
	blendNames := []string{
		"debian-accessibility", "debian-astro", "debian-edu", "debian-games",
		"debian-gis", "debian-hamradio", "debian-junior", "debian-med",
		"debian-multimedia", "debian-science", "freedombox",
	}

	fragments := []string{
		"kitchen/components/fragments/base-anchors.yaml",
		"kitchen/components/fragments/service-templates.yaml",
		"kitchen/components/fragments/blend-anchors.yaml",
	}

	for _, frag := range fragments {
		if _, err := os.Stat(filepath.Join(projectRoot, frag)); err != nil {
			return CheckResult{Status: "ok", Message: "fragments missing (skipped)"}, nil
		}
	}

	failed := 0
	total := len(blendNames)

	for _, blend := range blendNames {
		blendFile := filepath.Join(projectRoot, "kitchen", "recipes", blend+".yaml")
		if _, err := os.Stat(blendFile); err != nil {
			failed++
			continue
		}

		content, err := os.ReadFile(blendFile)
		if err != nil {
			failed++
			continue
		}

		if !strings.Contains(string(content), "kind: pure blend") {
			failed++
			continue
		}

		merged := ""
		for _, frag := range fragments {
			if fragContent, err := os.ReadFile(filepath.Join(projectRoot, frag)); err == nil {
				merged += string(fragContent) + "\n"
			}
		}
		merged += string(content)

		tmpFile, err := os.CreateTemp("", "blend-*.yaml")
		if err != nil {
			failed++
			continue
		}
		tmpFile.WriteString(merged)
		tmpFile.Close()

		fields := []string{".name", ".categories", ".variables"}
		for _, field := range fields {
			if _, err := runCmd(projectRoot, "yq", "eval", field, tmpFile.Name()); err != nil {
				failed++
				break
			}
		}
		os.Remove(tmpFile.Name())
	}

	if failed == 0 {
		return CheckResult{Status: "ok", Message: fmt.Sprintf("%d blends valid", total)}, nil
	}
	return CheckResult{Status: "warning", Message: fmt.Sprintf("%d/%d failed", failed, total)}, nil
}

func checkDocumentation(projectRoot string) (CheckResult, error) {
	requiredDocs := []string{"README.md", "ROADMAP.md", "CONTRIBUTING.md"}
	missing := []string{}

	for _, doc := range requiredDocs {
		if _, err := os.Stat(filepath.Join(projectRoot, doc)); err != nil {
			missing = append(missing, doc)
		}
	}

	if len(missing) == 0 {
		return CheckResult{Status: "ok"}, nil
	}
	return CheckResult{Status: "warning", Message: fmt.Sprintf("missing: %s", strings.Join(missing, ", "))}, nil
}
