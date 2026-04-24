package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().MarginLeft(2).Bold(true).Foreground(lipgloss.Color("205"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

// ListItem represents a selectable item
type ListItem struct {
	ItemTitle       string
	ItemDescription string
	Value           string
	Tags            []string
}

// FilterValue implements list.Item interface
func (i ListItem) FilterValue() string { return i.ItemTitle }

// Title returns the item title
func (i ListItem) Title() string { return i.ItemTitle }

// Description returns the item description
func (i ListItem) Description() string { return i.ItemDescription }

// Choice holds the selection result
type Choice struct {
	Item      ListItem
	Index     int
	Cancelled bool
}

// SelectModel is a single-select list model
type SelectModel struct {
	list   list.Model
	title  string
	width  int
	height int
	result *Choice
}

// NewSelect creates a new selection list
func NewSelect(title string, items []ListItem, height int) *SelectModel {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	l := list.New(listItems, list.NewDefaultDelegate(), 40, height)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	return &SelectModel{
		list:   l,
		title:  title,
		height: height,
		width:  40,
	}
}

// Init initializes the model
func (m SelectModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i, ok := m.list.SelectedItem().(ListItem); ok {
				m.result = &Choice{
					Item:      i,
					Index:     m.list.Index(),
					Cancelled: false,
				}
				return m, tea.Quit
			}
		case "esc", "ctrl+c":
			m.result = &Choice{Cancelled: true}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the model
func (m SelectModel) View() string {
	return "\n" + m.list.View()
}

// GetResult returns the selection result
func (m SelectModel) GetResult() *Choice {
	return m.result
}

// Select shows a single-select list and returns the choice
func Select(title string, items []ListItem, height int) (*Choice, error) {
	model := NewSelect(title, items, height)
	p := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}
	return finalModel.(SelectModel).GetResult(), nil
}

// ========== Confirm Dialog ==========

// ConfirmModel is a yes/no confirmation dialog
type ConfirmModel struct {
	question string
	choice   bool
}

// NewConfirm creates a new confirmation dialog
func NewConfirm(question string, defaultYes bool) *ConfirmModel {
	return &ConfirmModel{
		question: question,
		choice:   defaultYes,
	}
}

// Init initializes the model
func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.choice = true
			return m, tea.Quit
		case "n", "N":
			m.choice = false
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.choice = false
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the model
func (m ConfirmModel) View() string {
	yesStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	noStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

	var indicator string
	if m.choice {
		indicator = fmt.Sprintf("[%s] %s", yesStyle.Render("Y"), noStyle.Render("n"))
	} else {
		indicator = fmt.Sprintf("[%s] %s", yesStyle.Render("y"), noStyle.Render("N"))
	}

	return fmt.Sprintf("\n%s %s\n\n", m.question, indicator)
}

// Confirm shows a yes/no confirmation
func Confirm(question string, defaultYes bool) bool {
	model := NewConfirm(question, defaultYes)
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return false
	}
	return finalModel.(ConfirmModel).choice
}

// ========== Progress View ==========

// ProgressModel shows progress through steps
type ProgressModel struct {
	steps   []string
	current int
	title   string
	done    bool
	message string
}

// StepMsg represents a step completion
type StepMsg struct {
	Step    int
	Success bool
	Message string
}

// NewProgress creates a new progress view
func NewProgress(title string, steps []string) *ProgressModel {
	return &ProgressModel{
		steps: steps,
		title: title,
	}
}

// Init initializes the model
func (m ProgressModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StepMsg:
		if msg.Success {
			m.current = msg.Step + 1
		}
		m.message = msg.Message
		if m.current >= len(m.steps) {
			m.done = true
		}
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the model
func (m ProgressModel) View() string {
	if m.done {
		return fmt.Sprintf("\n%s Complete!\n\n", m.title)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "\n%s\n\n", m.title)

	checkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	pendingStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	for i, step := range m.steps {
		if i < m.current {
			fmt.Fprintf(&sb, "  %s %s\n", checkStyle.Render("✓"), step)
		} else if i == m.current {
			fmt.Fprintf(&sb, "  %s %s\n", activeStyle.Render("▶"), step)
		} else {
			fmt.Fprintf(&sb, "  %s %s\n", pendingStyle.Render("○"), step)
		}
	}

	if m.message != "" {
		fmt.Fprintf(&sb, "\n  %s\n", m.message)
	}

	return sb.String()
}

// Step advances to the next step with a tea.Cmd
func Step(m *ProgressModel, step int, success bool, message string) tea.Cmd {
	return func() tea.Msg {
		return StepMsg{Step: step, Success: success, Message: message}
	}
}

// ========== Info Viewer ==========

// InfoModel displays detailed information
type InfoModel struct {
	viewport viewport.Model
	title    string
}

// NewInfo creates a new info viewer
func NewInfo(title, content string, height int) *InfoModel {
	v := viewport.New(80, height)
	v.SetContent(content)

	return &InfoModel{
		viewport: v,
		title:    title,
	}
}

// Init initializes the model
func (m InfoModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m InfoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 4
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View renders the model
func (m InfoModel) View() string {
	title := titleStyle.Render(m.title)
	return fmt.Sprintf("\n%s\n\n%s\n\n%s", title, m.viewport.View(), helpStyle.Render("q: quit | ↑/↓: scroll"))
}

// ShowInfo displays information in a scrollable view
func ShowInfo(title, content string, height int) error {
	model := NewInfo(title, content, height)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// ========== Simple Input ==========

// InputModel is a simple text input
type InputModel struct {
	value       string
	placeholder string
	title       string
	submitted   bool
}

// NewInput creates a new text input
func NewInput(title, placeholder, defaultValue string) *InputModel {
	return &InputModel{
		title:       title,
		placeholder: placeholder,
		value:       defaultValue,
	}
}

// Init initializes the model
func (m InputModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.submitted = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.value = ""
			return m, tea.Quit
		case "backspace":
			if len(m.value) > 0 {
				m.value = m.value[:len(m.value)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.value += msg.String()
			}
		}
	}
	return m, nil
}

// View renders the model
func (m InputModel) View() string {
	inputStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)

	val := m.value
	if val == "" {
		val = m.placeholder
		val = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(val)
	}

	return fmt.Sprintf("\n%s\n\n%s %s\n\n",
		m.title,
		inputStyle.Render(val),
		helpStyle.Render("enter: confirm | esc: cancel"))
}

// GetValue returns the input value
func (m InputModel) GetValue() string {
	return m.value
}

// WasSubmitted returns true if the user submitted
func (m InputModel) WasSubmitted() bool {
	return m.submitted
}

// GetInput shows a text input and returns the value
func GetInput(title, placeholder, defaultValue string) (string, bool) {
	model := NewInput(title, placeholder, defaultValue)
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return "", false
	}
	m := finalModel.(InputModel)
	return m.GetValue(), m.WasSubmitted()
}
