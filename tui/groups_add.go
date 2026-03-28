package tui

import (
	"github.com/jprincevevo/reap/config"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

// groupAddModel is a multi-select list used when creating a new group to pick
// which repos belong to it. It is a type alias for selectListModel so all
// selectable-list behaviour (space toggle, ctrl+a select all, ctrl+d deselect
// all, enter confirm) is provided automatically.
type groupAddModel = selectListModel

// NewGroupAddModel returns a ready-to-use group-add selectable list populated
// with every repo in the config, all starting unselected.
func NewGroupAddModel(cfg *config.Config) groupAddModel {
	var items []list.Item
	for _, repo := range cfg.Repos {
		items = append(items, repoItem{url: repo.URL, selected: false})
	}
	return newSelectList(items, repoDelegate{}, "Select repositories to add to the group")
}

// InitialGroupAddModel runs a standalone program and returns the URLs of the
// repos the user selected.
func InitialGroupAddModel(cfg *config.Config) ([]string, error) {
	m := NewGroupAddModel(cfg)

	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	var selected []string
	if m, ok := finalModel.(selectListModel); ok {
		for _, listItem := range m.Items() {
			if i, ok := listItem.(repoItem); ok && i.selected {
				selected = append(selected, i.url)
			}
		}
	}

	return selected, nil
}
