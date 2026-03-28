package tui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type promptModel struct {
	input    textinput.Model
	title    string
	quitting bool
	ready    bool
}

func (m promptModel) Init() tea.Cmd {
	return m.input.Focus()
}

func (m promptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ready = true
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.input.SetValue("")
			m.quitting = true
			return m, tea.Quit
		case "enter":
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m promptModel) View() tea.View {
	if !m.ready {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}
	if m.quitting {
		return tea.NewView("")
	}
	content := "\n" + titleStyle.Render(m.title) + "\n\n    " + m.input.View() + "\n"
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func NewPromptModel(title, placeholder string) promptModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Prompt = "  > "
	ti.SetWidth(60)
	return promptModel{input: ti, title: title}
}

func NewPromptModelWithValue(title, placeholder, value string) promptModel {
	m := NewPromptModel(title, placeholder)
	m.input.SetValue(value)
	return m
}

func InitialPromptModel(title, placeholder string) (string, error) {
	m := NewPromptModel(title, placeholder)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}
	if fm, ok := finalModel.(promptModel); ok {
		return fm.input.Value(), nil
	}
	return "", nil
}
