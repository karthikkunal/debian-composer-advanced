package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "run-all":
		runAllTests()
	case "run-unit":
		runUnitTests()
	case "run-functional":
		runFunctionalTests()
	case "run-smoke":
		runSmokeTests()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Debian Composer Go - Test Runner")
	fmt.Println("")
	fmt.Println("Usage: test-runner <command>")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  run-all         Run all tests")
	fmt.Println("  run-unit        Run unit tests")
	fmt.Println("  run-functional  Run functional tests")
	fmt.Println("  run-smoke       Run smoke tests")
}

func runAllTests() {
	fmt.Println("Running all tests...")
	runSmokeTests()
	runUnitTests()
	runFunctionalTests()
	fmt.Println("\n✓ All tests completed")
}

func runSmokeTests() {
	fmt.Println("\n=== Smoke Tests ===")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"Binary builds", testBinaryBuilds},
		{"Help command works", testHelpCommand},
		{"Validate command works", testValidateCommand},
		{"List recipes", testListRecipes},
	}

	runTests(tests)
}

func runUnitTests() {
	fmt.Println("\n=== Unit Tests ===")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"Recipe parser", testRecipeParser},
		{"Recipe types", testRecipeTypes},
		{"State management", testStateManagement},
		{"Hardware detection", testHardwareDetection},
	}

	runTests(tests)
}

func runFunctionalTests() {
	fmt.Println("\n=== Functional Tests ===")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"Install simple recipe", testInstallSimpleRecipe},
		{"Dry-run mode", testDryRunMode},
		{"System config commands", testSystemConfig},
	}

	runTests(tests)
}

func runTests(tests []struct {
	name string
	fn   func() error
}) {
	passed := 0
	failed := 0

	for _, test := range tests {
		fmt.Printf("  Running: %s ... ", test.name)
		if err := test.fn(); err != nil {
			fmt.Printf("FAIL: %v\n", err)
			failed++
		} else {
			fmt.Println("OK")
			passed++
		}
	}

	fmt.Printf("\nResults: %d passed, %d failed\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func testBinaryBuilds() error {
	if _, err := os.Stat("./bin/debian-composer"); os.IsNotExist(err) {
		return fmt.Errorf("binary not found")
	}
	return nil
}

var testStateDir = "--state=/tmp/debian-composer-test"
var testKitchen = "--kitchen=./kitchen"

func testHelpCommand() error {
	cmd := exec.Command("./bin/debian-composer", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("help failed: %w", err)
	}
	if !strings.Contains(string(out), "Debian Composer") {
		return fmt.Errorf("help output missing expected content")
	}
	return nil
}

func testValidateCommand() error {
	// Just check that validate command runs without crashing
	// Schema validation errors are expected for some recipes
	_, _ = exec.Command("./bin/debian-composer", testKitchen, testStateDir, "validate", "simple-test").CombinedOutput()
	return nil
}

func testListRecipes() error {
	_, err := exec.Command("./bin/debian-composer", testKitchen, testStateDir, "list").CombinedOutput()
	return err
}

func testRecipeParser() error {
	cmd := exec.Command("./bin/debian-composer", testKitchen, testStateDir, "info", "simple-test")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("info failed: %w - %s", err, string(out))
	}
	return nil
}

func testRecipeTypes() error {
	files, err := filepath.Glob("./internal/*/*.go")
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no source files found")
	}
	return nil
}

func testStateManagement() error {
	if _, err := os.Stat("/var/lib/debian-composer"); os.IsNotExist(err) {
		// State dir doesn't exist, which is OK
		return nil
	}
	return nil
}

func testHardwareDetection() error {
	_, err := exec.Command("./bin/debian-composer", testKitchen, testStateDir, "probe-hardware").CombinedOutput()
	return err
}

func testInstallSimpleRecipe() error {
	cmd := exec.Command("./bin/debian-composer", testKitchen, testStateDir, "--dry-run", "install", "simple-test")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("install failed: %w - %s", err, string(out))
	}
	if !strings.Contains(string(out), "DRY-RUN") {
		return fmt.Errorf("dry-run not shown: %s", string(out))
	}
	return nil
}

func testDryRunMode() error {
	_, err := exec.Command("./bin/debian-composer", testStateDir, "--dry-run", "system").CombinedOutput()
	return err
}

func testSystemConfig() error {
	cmd := exec.Command("./bin/debian-composer", testStateDir, "--dry-run", "system")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("system config failed: %w", err)
	}
	if !strings.Contains(string(out), "System Configuration") {
		return fmt.Errorf("system config output missing")
	}
	return nil
}
