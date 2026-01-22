package tui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type cloneMsg struct{ repo string }
type errMsg struct{ err error }

type cloneModel struct {
	spinner  spinner.Model
	repos    []string
	cloned   []string
	errors   []error
	quitting bool
}

func (m cloneModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.cloneNext)
}

func (m cloneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	case cloneMsg:
		m.cloned = append(m.cloned, msg.repo)
		return m, m.cloneNext
	case errMsg:
		m.errors = append(m.errors, msg.err)
		return m, m.cloneNext
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m cloneModel) View() string {
	if m.quitting {
		return "Aborting...\n"
	}

	var s strings.Builder
	s.WriteString("Cloning repositories...\n\n")
	s.WriteString(m.spinner.View() + " Cloning...\n\n")
	for _, repo := range m.cloned {
		s.WriteString(fmt.Sprintf("✓ %s\n", repo))
	}
	for _, err := range m.errors {
		s.WriteString(fmt.Sprintf("✗ %s\n", err))
	}
	return s.String()
}

func (m *cloneModel) cloneNext() tea.Msg {
	if len(m.repos) == 0 {
		return tea.Quit()
	}

	repo := m.repos[0]
	m.repos = m.repos[1:]

	cmd := exec.Command("git", "clone", repo)
	if err := cmd.Run(); err != nil {
		return errMsg{err}
	}

	return cloneMsg{repo}
}

func NewCloneModel(repos []string) cloneModel {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return cloneModel{
		spinner: s,
		repos:   repos,
	}
}

func InitialCloneModel(repos []string) error {
	m := NewCloneModel(repos)
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}
