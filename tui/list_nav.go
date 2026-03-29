package tui

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

// navListEvent is the outcome of a navListModel.Update call that the parent
// should react to.
type navListEvent uint8

const (
	navListEventNone   navListEvent = iota
	navListEventEnter               // enter pressed (not while filtering)
	navListEventGoBack              // esc pressed when no filter is active
	navListEventQuit                // ctrl+c or q pressed
)

// navListModel wraps list.Model with standard navigation boilerplate.
//
// It embeds list.Model so all list methods (Items, FilterState, SetSize,
// SetFilterText, VisibleItems, SelectedItem, SetFilteringEnabled,
// AdditionalShortHelpKeys, Title, …) are promoted and accessible directly.
//
// Update signature is (navListModel, navListEvent, tea.Cmd) — it is NOT a
// tea.Model. Parents call it, inspect the returned event, and handle their
// own domain-specific keys before forwarding unrecognised messages here.
//
// Handled automatically:
//   - tea.WindowSizeMsg → SetSize(w, listHeight)
//   - ctrl+c / q        → navListEventQuit  + tea.Quit
//   - esc (filtering)   → clear filter, navListEventNone
//   - esc (idle)        → navListEventGoBack (no tea.Quit; parent decides)
//   - enter             → navListEventEnter  (no tea.Quit; parent decides)
//   - everything else   → forwarded to list.Model.Update
type navListModel struct {
	list.Model
}

// newNavList constructs a navListModel with the shared style applied.
func newNavList(items []list.Item, delegate list.ItemDelegate, title string) navListModel {
	return navListModel{Model: newList(items, delegate, title)}
}

// SetSize overrides list.Model.SetSize to ensure ShowFullHelp stays disabled
// after every resize, since list.Model.SetSize unconditionally re-enables it.
func (m *navListModel) SetSize(w, h int) {
	m.Model.SetSize(w, h)
	m.Model.KeyMap.ShowFullHelp.SetEnabled(false)
	m.Model.KeyMap.CloseFullHelp.SetEnabled(false)
}

// Update processes msg and returns the updated model, an event for the parent
// to act on, and any command. Parents should handle domain-specific keys
// (a, r, d, …) before calling Update and return early if they fire.
func (m navListModel) Update(msg tea.Msg) (navListModel, navListEvent, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, listHeight)
		return m, navListEventNone, nil

	case tea.KeyPressMsg:
		if m.Model.FilterState() != list.Filtering {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, navListEventQuit, tea.Quit
			case "esc":
				if m.Model.FilterState() == list.FilterApplied {
					m.Model.ResetFilter()
					return m, navListEventNone, nil
				}
				return m, navListEventGoBack, nil
			case "enter":
				return m, navListEventEnter, nil
			}
		}
	}

	var cmd tea.Cmd
	m.Model, cmd = m.Model.Update(msg)
	return m, navListEventNone, cmd
}
