package snapper

import (
	"testing"
)

func TestIsAvailable(t *testing.T) {
	available := IsAvailable()
	if !available {
		t.Log("Snapper is not installed - installation instructions would be provided")
		t.Log("Install with: sudo apt install snapper")
	}
}

func TestNew(t *testing.T) {
	mgr := New(false)
	if mgr == nil {
		t.Error("Expected manager, got nil")
		return
	}
	if mgr.config != "root" {
		t.Errorf("Expected default config 'root', got '%s'", mgr.config)
	}
}

func TestNewWithConfig(t *testing.T) {
	mgr := New(false, "home")
	if mgr.config != "home" {
		t.Errorf("Expected config 'home', got '%s'", mgr.config)
	}
}

func TestDryRun(t *testing.T) {
	mgr := New(true)
	snap, err := mgr.CreateSnapshot("single", "test snapshot")
	if err != nil {
		t.Errorf("Dry run should not error: %v", err)
	}
	if snap.ID != -1 {
		t.Errorf("Expected dry-run ID -1, got %d", snap.ID)
	}
	if snap.Name != "dry-run-snapshot" {
		t.Errorf("Expected dry-run name, got '%s'", snap.Name)
	}
}

func TestCheckAvailable(t *testing.T) {
	available, message := CheckAvailable()
	if available {
		t.Logf("Snapper available: %s", message)
	} else {
		t.Logf("Snapper not available: %s", message)
	}
}
