package tui

import (
	"fmt"

	"github.com/jprincevevo/reap/config"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

type removeModel struct {
	list     list.Model
	choice   string
	quitting bool
	ready    bool
}

func (m removeModel) Init() tea.Cmd {
	return nil
}

func (m removeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, listHeight)
		m.ready = true
		return m, nil

	case tea.KeyPressMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				m.choice = string(i)
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m removeModel) View() tea.View {
	if !m.ready {
		return tea.NewView("")
	}
	if m.choice != "" {
		return tea.NewView(quitTextStyle.Render(fmt.Sprintf("Selected repository: %s", m.choice)))
	}
	if m.quitting {
		return tea.NewView(quitTextStyle.Render("Cancelling..."))
	}
	v := tea.NewView("\n" + m.list.View())
	v.AltScreen = true
	return v
}

func NewRemoveModel(cfg *config.Config) removeModel {
	var items []list.Item
	for _, repo := range cfg.Repos {
		items = append(items, item(repo.URL))
	}

	l := newList(items, itemDelegate{}, "Select a repository to remove")
	l.SetFilteringEnabled(false) // removal list intentionally has no filter

	return removeModel{list: l}
}

func InitialRemoveModel(cfg *config.Config) (string, error) {
	m := NewRemoveModel(cfg)

	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	if m, ok := finalModel.(removeModel); ok && m.choice != "" {
		return m.choice, nil
	}

	return "", fmt.Errorf("no repository selected")
}
