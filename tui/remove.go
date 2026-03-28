package tui

import (
	"fmt"

	"github.com/jprincevevo/reap/config"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

type removeModel struct {
	list     navListModel
	choice   string
	quitting bool
	ready    bool
}

func (m removeModel) Init() tea.Cmd {
	return nil
}

func (m removeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		m.ready = true
	}

	var event navListEvent
	var cmd tea.Cmd
	m.list, event, cmd = m.list.Update(msg)

	switch event {
	case navListEventQuit:
		m.quitting = true
		return m, cmd
	case navListEventEnter:
		if i, ok := m.list.SelectedItem().(item); ok {
			m.choice = string(i)
		}
		return m, tea.Quit
	}

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

	l := newNavList(items, itemDelegate{}, "Select a repository to remove")
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
