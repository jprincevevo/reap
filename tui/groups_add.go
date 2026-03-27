package tui

import (
	"github.com/jprincevevo/reap/config"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

type groupAddModel struct {
	list     list.Model
	quitting bool
	ready    bool
}

func (m groupAddModel) Init() tea.Cmd {
	return nil
}

func (m groupAddModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return m, tea.Quit

		case "space":
			if i, ok := m.list.SelectedItem().(repoItem); ok {
				i.selected = !i.selected
				cmd := m.list.SetItem(m.list.Index(), i)
				return m, cmd
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m groupAddModel) View() tea.View {
	if !m.ready {
		return tea.NewView("")
	}
	if m.quitting {
		return tea.NewView(quitTextStyle.Render("Cancelling..."))
	}
	v := tea.NewView("\n" + m.list.View())
	v.AltScreen = true
	return v
}

func NewGroupAddModel(cfg *config.Config) groupAddModel {
	var items []list.Item
	for _, repo := range cfg.Repos {
		items = append(items, repoItem{url: repo.URL, selected: false})
	}

	const defaultWidth = 20

	l := list.New(items, repoDelegate{}, defaultWidth, listHeight)
	l.Title = "Select repositories to add to the group"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	return groupAddModel{list: l}
}

func InitialGroupAddModel(cfg *config.Config) ([]string, error) {
	m := NewGroupAddModel(cfg)

	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	var selected []string
	if m, ok := finalModel.(groupAddModel); ok {
		for _, item := range m.list.Items() {
			if i, ok := item.(repoItem); ok && i.selected {
				selected = append(selected, i.url)
			}
		}
	}

	return selected, nil
}
