package ux

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/olekukonko/tablewriter"
)

var (
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

// Spinner wraps charmbracelet/bubbles spinner frames for loading states
type Spinner struct {
	frames []string
	msg    string
	done   chan struct{}
	mu     sync.Mutex
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		frames: spinner.Dot.Frames,
		msg:    message,
		done:   make(chan struct{}),
	}
}

// Start starts the spinner in a background goroutine
func (s *Spinner) Start() {
	go func() {
		frame := 0
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				s.mu.Lock()
				fmt.Fprintf(os.Stderr, "\r%s %s   ", spinnerStyle.Render(s.frames[frame%len(s.frames)]), s.msg)
				frame++
				s.mu.Unlock()
			}
		}
	}()
}

// Stop stops the spinner with a message
func (s *Spinner) Stop(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	fmt.Fprintf(os.Stderr, "\r%-80s\r%s\n", "", message)
}

// StopWithError stops the spinner with an error message
func (s *Spinner) StopWithError(message string) {
	s.Stop(errorStyle.Render(fmt.Sprintf("✗ %s", message)))
}

// StopWithSuccess stops the spinner with a success message
func (s *Spinner) StopWithSuccess(message string) {
	s.Stop(successStyle.Render(fmt.Sprintf("✓ %s", message)))
}

// UpdateMessage updates the spinner message
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msg = message
}

// Progress wraps charmbracelet/bubbles progress bar
type Progress struct {
	total   int
	current int
	desc    string
	bar     progress.Model
}

// NewProgress creates a new progress bar
func NewProgress(total int, description string) *Progress {
	return &Progress{
		total: total,
		desc:  description,
		bar:   progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage()),
	}
}

// Add increments the progress bar
func (p *Progress) Add(n int) {
	p.current += n
	p.render()
}

// Finish completes the progress bar
func (p *Progress) Finish() {
	p.current = p.total
	p.render()
	fmt.Fprintln(os.Stderr)
}

// Set sets the current progress value
func (p *Progress) Set(current int) {
	p.current = current
	p.render()
}

func (p *Progress) render() {
	var pct float64
	if p.total > 0 {
		pct = float64(p.current) / float64(p.total)
		if pct > 1 {
			pct = 1
		}
	}
	fmt.Fprintf(os.Stderr, "\r%s %s", p.desc, p.bar.ViewAs(pct))
}

// MultiProgress manages multiple progress bars
type MultiProgress struct {
	mu   sync.Mutex
	bars []*Progress
}

// NewMultiProgress creates a new multi-progress manager
func NewMultiProgress() *MultiProgress {
	return &MultiProgress{}
}

// Add adds a new progress bar and returns its index
func (m *MultiProgress) Add(total int, description string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	bar := NewProgress(total, description)
	idx := len(m.bars)
	m.bars = append(m.bars, bar)
	return idx
}

// Update updates a specific progress bar
func (m *MultiProgress) Update(idx int, current int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if idx >= 0 && idx < len(m.bars) {
		m.bars[idx].Set(current)
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
