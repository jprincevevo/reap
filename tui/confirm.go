package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 0).
			BorderTop(true).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true)

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.Color("#888B7E")).
			Padding(0, 3).
			MarginTop(1)

	activeButtonStyle = buttonStyle.Copy().
				Foreground(lipgloss.Color("#FFF7DB")).
				Background(lipgloss.Color("#F25D94")).
				Underline(true)
)

type confirmModel struct {
	prompt      string
	confirmed   bool
	quitting    bool
	activeButton int
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "left", "h":
			if m.activeButton > 0 {
				m.activeButton--
			}
		case "right", "l":
			if m.activeButton < 1 {
				m.activeButton++
			}
		case "enter":
			if m.activeButton == 0 {
				m.confirmed = true
			}
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	if m.quitting {
		return ""
	}

	var yes, no string
	if m.activeButton == 0 {
		yes = activeButtonStyle.Render("Yes")
		no = buttonStyle.Render("No")
	} else {
		yes = buttonStyle.Render("Yes")
		no = activeButtonStyle.Render("No")
	}

	question := lipgloss.NewStyle().Width(50).Align(lipgloss.Center).Render(m.prompt)
	buttons := lipgloss.JoinHorizontal(lipgloss.Top, yes, no)
	ui := lipgloss.JoinVertical(lipgloss.Center, question, buttons)

	return dialogBoxStyle.Render(ui)
}

func InitialConfirmModel(prompt string) (bool, error) {
	m := confirmModel{prompt: prompt}
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}

	if m, ok := finalModel.(confirmModel); ok {
		return m.confirmed, nil
	}

	return false, fmt.Errorf("could not determine confirmation")
}
