package ux

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/olekukonko/tablewriter"
	"github.com/schollz/progressbar/v3"
)

// Spinner wraps the spinner library for loading states
type Spinner struct {
	spinner *spinner.Spinner
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message
	s.Color("cyan")
	return &Spinner{spinner: s}
}

// Start starts the spinner
func (s *Spinner) Start() {
	s.spinner.Start()
}

// Stop stops the spinner with a message
func (s *Spinner) Stop(message string) {
	s.spinner.FinalMSG = message + "\n"
	s.spinner.Stop()
}

// StopWithError stops the spinner with an error message
func (s *Spinner) StopWithError(message string) {
	s.spinner.Color("red")
	s.spinner.FinalMSG = fmt.Sprintf("✗ %s\n", message)
	s.spinner.Stop()
}

// StopWithSuccess stops the spinner with a success message
func (s *Spinner) StopWithSuccess(message string) {
	s.spinner.Color("green")
	s.spinner.FinalMSG = fmt.Sprintf("✓ %s\n", message)
	s.spinner.Stop()
}

// UpdateMessage updates the spinner message
func (s *Spinner) UpdateMessage(message string) {
	s.spinner.Suffix = " " + message
}

// Progress wraps the progressbar library for task progress
type Progress struct {
	bar *progressbar.ProgressBar
}

// NewProgress creates a new progress bar
func NewProgress(total int, description string) *Progress {
	bar := progressbar.NewOptions(total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetElapsedTime(false),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
	)
	return &Progress{bar: bar}
}

// Add increments the progress bar
func (p *Progress) Add(n int) {
	p.bar.Add(n)
}

// Finish completes the progress bar
func (p *Progress) Finish() {
	p.bar.Finish()
}

// Set sets the current progress value
func (p *Progress) Set(current int) {
	p.bar.Set(current)
}

// MultiProgress manages multiple progress bars
type MultiProgress struct {
	mu      sync.Mutex
	bars    []*Progress
	current int
}

// NewMultiProgress creates a new multi-progress manager
func NewMultiProgress() *MultiProgress {
	return &MultiProgress{}
}

// Add adds a new progress bar
func (m *MultiProgress) Add(total int, description string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	bar := NewProgress(total, description)
	idx := len(m.bars)
	m.bars = append(m.bars, bar)
	return idx
}

// Update updates a specific progress bar
func (m *MultiProgress) Update(idx int, progress int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if idx >= 0 && idx < len(m.bars) {
		m.bars[idx].Set(progress)
	}
}

// Finish finishes all progress bars
func (m *MultiProgress) Finish() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, bar := range m.bars {
		bar.Finish()
	}
}

// Logger provides formatted output for CLI
type Logger struct {
	verbose bool
	writer  io.Writer
}

// NewLogger creates a new logger
func NewLogger(verbose bool) *Logger {
	return &Logger{
		verbose: verbose,
		writer:  os.Stdout,
	}
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.writer, "%s\n", msg)
}

// Infof logs an info message with formatting
func (l *Logger) Infof(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.writer, "  %s\n", msg)
}

// Verbose logs a verbose message (only shown when verbose is enabled)
func (l *Logger) Verbose(format string, args ...interface{}) {
	if l.verbose {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(l.writer, "  [verbose] %s\n", msg)
	}
}

// Success logs a success message
func (l *Logger) Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.writer, "\x1b[32m✓ %s\x1b[0m\n", msg)
}

// Warning logs a warning message
func (l *Logger) Warning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.writer, "\x1b[33m⚠ %s\x1b[0m\n", msg)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "\x1b[31m✗ %s\x1b[0m\n", msg)
}

// Header logs a section header
func (l *Logger) Header(text string) {
	fmt.Fprintf(l.writer, "\n\x1b[1m%s\x1b[0m\n", text)
	fmt.Fprintf(l.writer, "%s\n", strings.Repeat("─", len(text)))
}

// Step logs a numbered step
func (l *Logger) Step(num int, total int, text string) {
	fmt.Fprintf(l.writer, "  [%d/%d] %s\n", num, total, text)
}

// DryRun logs a dry-run message
func (l *Logger) DryRun(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.writer, "\x1b[36m[dry-run] %s\x1b[0m\n", msg)
}

// PrintTable prints a formatted table using tablewriter
func PrintTable(headers []string, rows [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetBorder(false)
	table.SetColumnSeparator("  ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	for _, row := range rows {
		table.Append(row)
	}
	table.Render()
}

// Confirm prompts the user for confirmation
func Confirm(message string, defaultYes bool) bool {
	defaultStr := "Y/n"
	if !defaultYes {
		defaultStr = "y/N"
	}
	fmt.Printf("%s [%s]: ", message, defaultStr)

	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))

	if response == "" {
		return defaultYes
	}
	return response == "y" || response == "yes"
}
