package ux

import (
	"bytes"
	"testing"
)

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		verbose: true,
		writer:  &buf,
	}

	logger.Info("test message")
	if buf.String() == "" {
		t.Error("Info() wrote nothing")
	}
}

func TestLoggerVerbose(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		verbose: false,
		writer:  &buf,
	}

	logger.Verbose("should not appear")
	if buf.Len() > 0 {
		t.Error("Verbose() wrote when verbose=false")
	}

	buf.Reset()
	logger.verbose = true
	logger.Verbose("should appear")
	if buf.Len() == 0 {
		t.Error("Verbose() wrote nothing when verbose=true")
	}
}

func TestLoggerSuccess(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		writer: &buf,
	}

	logger.Success("success message")
	output := buf.String()
	if output == "" {
		t.Error("Success() wrote nothing")
	}
	// Should contain the checkmark
	if len(output) < 10 {
		t.Error("Success() output too short")
	}
}

func TestLoggerError(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		writer: &buf,
	}

	// Error writes to stderr, so we can't easily capture it
	// Just verify it doesn't panic
	logger.Error("error message")
}

func TestPrintTable(t *testing.T) {
	headers := []string{"Name", "Value", "Status"}
	rows := [][]string{
		{"foo", "123", "ok"},
		{"bar", "456", "error"},
		{"baz", "789", "ok"},
	}

	// Just verify it doesn't panic
	PrintTable(headers, rows)
}

func TestPrintTableEmpty(t *testing.T) {
	headers := []string{"Name", "Value"}
	rows := [][]string{}

	// Should handle empty rows
	PrintTable(headers, rows)
}

func TestPrintTableUnevenRows(t *testing.T) {
	headers := []string{"A", "B", "C"}
	rows := [][]string{
		{"1", "2"},
		{"3", "4", "5", "6"},
	}

	// Should handle uneven rows
	PrintTable(headers, rows)
}

func TestNewSpinner(t *testing.T) {
	s := NewSpinner("loading")
	if s == nil {
		t.Fatal("NewSpinner() returned nil")
	}
	if s.program == nil {
		t.Fatal("NewSpinner() program is nil")
	}
}

func TestNewProgress(t *testing.T) {
	bar := NewProgress(100, "test")
	if bar == nil {
		t.Fatal("NewProgress() returned nil")
	}
	if bar.total != 100 {
		t.Fatalf("NewProgress() total = %d, want 100", bar.total)
	}
}

func TestNewMultiProgress(t *testing.T) {
	mp := NewMultiProgress()
	if mp == nil {
		t.Fatal("NewMultiProgress() returned nil")
	}

	idx := mp.Add(100, "test")
	if idx != 0 {
		t.Errorf("Expected index 0, got %d", idx)
	}

	mp.Update(0, 50)
	mp.Finish()
}
