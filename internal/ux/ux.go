package ux

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	charmlog "github.com/charmbracelet/log"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/olekukonko/tablewriter"
)

var (
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

// spinnerModel is the bubbletea model for the Spinner.
type spinnerModel struct {
	spinner spinner.Model
	msg     string
	done    bool
}

type spinnerStopMsg struct{ finalMsg string }
type spinnerUpdateMsg struct{ msg string }

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinnerStopMsg:
		m.done = true
		m.msg = msg.finalMsg
		return m, tea.Quit
	case spinnerUpdateMsg:
		m.msg = msg.msg
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m spinnerModel) View() string {
	if m.done {
		return ""
	}
	return spinnerStyle.Render(m.spinner.View()) + " " + m.msg
}

// Spinner provides an animated loading indicator using the bubbletea model loop.
type Spinner struct {
	program *tea.Program
	done    chan struct{}
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	m := spinnerModel{
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(spinnerStyle)),
		msg:     message,
	}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithInput(nil))
	return &Spinner{program: p, done: make(chan struct{})}
}

// Start starts the spinner in a background goroutine
func (s *Spinner) Start() {
	go func() {
		defer close(s.done)
		s.program.Run() //nolint:errcheck
	}()
}

// Stop stops the spinner and prints a final message
func (s *Spinner) Stop(message string) {
	s.program.Send(spinnerStopMsg{finalMsg: message})
	<-s.done
	fmt.Fprintf(os.Stderr, "%s\n", message)
}

// StopWithError stops the spinner with an error message
func (s *Spinner) StopWithError(message string) {
	s.Stop(errorStyle.Render(fmt.Sprintf("✗ %s", message)))
}

// StopWithSuccess stops the spinner with a success message
func (s *Spinner) StopWithSuccess(message string) {
	s.Stop(successStyle.Render(fmt.Sprintf("✓ %s", message)))
}

// UpdateMessage updates the spinner message while it is running
func (s *Spinner) UpdateMessage(message string) {
	s.program.Send(spinnerUpdateMsg{msg: message})
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

var (
	logSuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	logWarnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	logErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	logDryRunStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	logHeaderStyle  = lipgloss.NewStyle().Bold(true)
)

// Logger provides formatted output for CLI backed by charmbracelet/log.
type Logger struct {
	verbose bool
	writer  io.Writer
	log     *charmlog.Logger
}

// ensureLog lazily initialises the charmbracelet/log logger the first time it is needed.
// This allows Logger to be constructed as a struct literal in tests (&Logger{writer: &buf})
// without requiring a call to NewLogger.
func (l *Logger) ensureLog() *charmlog.Logger {
	if l.log != nil {
		return l.log
	}
	w := l.writer
	if w == nil {
		w = os.Stdout
	}
	styles := charmlog.DefaultStyles()
	styles.Levels[charmlog.InfoLevel] = lipgloss.NewStyle()
	styles.Levels[charmlog.DebugLevel] = lipgloss.NewStyle().SetString("[verbose]").Foreground(lipgloss.Color("8"))
	styles.Levels[charmlog.WarnLevel] = logWarnStyle.SetString("⚠")
	styles.Levels[charmlog.ErrorLevel] = logErrorStyle.SetString("✗")
	styles.Keys = map[string]lipgloss.Style{}
	styles.Values = map[string]lipgloss.Style{}
	l.log = charmlog.NewWithOptions(w, charmlog.Options{
		Formatter:       charmlog.TextFormatter,
		ReportTimestamp: false,
		Level:           charmlog.DebugLevel, // gating for Verbose is done in Go, not by level
	})
	l.log.SetStyles(styles)
	return l.log
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
	l.ensureLog().Infof(format, args...)
}

// Infof logs an indented info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.ensureLog().Infof("  "+format, args...)
}

// Verbose logs a message only when verbose mode is enabled
func (l *Logger) Verbose(format string, args ...interface{}) {
	if l.verbose {
		l.ensureLog().Debugf(format, args...)
	}
}

// Success logs a success message styled with a checkmark
func (l *Logger) Success(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.writer, "%s\n", logSuccessStyle.Render("✓ "+msg))
}

// Warning logs a warning message
func (l *Logger) Warning(format string, args ...interface{}) {
	l.ensureLog().Warnf(format, args...)
}

// Error logs an error message to stderr
func (l *Logger) Error(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "%s\n", logErrorStyle.Render("✗ "+msg))
}

// Header logs a bold section header with a separator line
func (l *Logger) Header(text string) {
	w := l.writer
	if w == nil {
		w = os.Stdout
	}
	fmt.Fprintf(w, "\n%s\n%s\n", logHeaderStyle.Render(text), strings.Repeat("─", len(text)))
}

// Step logs a numbered step
func (l *Logger) Step(num int, total int, text string) {
	w := l.writer
	if w == nil {
		w = os.Stdout
	}
	fmt.Fprintf(w, "  [%d/%d] %s\n", num, total, text)
}

// DryRun logs a dry-run preview message
func (l *Logger) DryRun(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	w := l.writer
	if w == nil {
		w = os.Stdout
	}
	fmt.Fprintf(w, "%s\n", logDryRunStyle.Render("[dry-run] "+msg))
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
