package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

// selectable is implemented by any list item that carries a boolean selected
// state. selectListModel uses this interface for space-toggle, ctrl+a
// (select all), and ctrl+d (deselect all), so the same component works for
// both repoItem and groupSelectItem without type-specific logic.
type selectable interface {
	list.Item
	isSelected() bool
	withSelected(bool) list.Item
}

// selectListModel is a multi-select list that implements tea.Model.
//
// It embeds list.Model so all list methods (Items, VisibleItems, FilterState,
// SetSize, SetFilterText, AdditionalShortHelpKeys, …) are promoted and
// accessible directly.
//
// Key bindings:
//   - space   → toggle current item's selected state
//   - ctrl+a  → select all items
//   - ctrl+d  → deselect all items
//   - enter   → set done = true, return tea.Quit
//   - esc     → clear active filter, or set goBack = true + tea.Quit
//   - q / ctrl+c → set quitting = true, return tea.Quit
//   - WindowSizeMsg → SetSize(w, listHeight), set ready = true
//
// After Update, inspect done / quitting / goBack to determine next action.
// Collect results via Items() (all) or VisibleItems() (filter-aware).
// Items must implement the selectable interface for toggle/select-all to work.
type selectListModel struct {
	list.Model
	ready    bool
	done     bool
	quitting bool
	goBack   bool
}

// newSelectList constructs a selectListModel with the shared style and
// standard help keys (space toggle, ctrl+a select all, ctrl+d deselect all,
// enter confirm). The delegate controls how items are rendered; pass
// repoDelegate{} for repo lists or groupSelectDelegate{} for group lists.
// Callers may override AdditionalShortHelpKeys after construction.
func newSelectList(items []list.Item, delegate list.ItemDelegate, title string) selectListModel {
	l := newList(items, delegate, title)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle")),
			key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("ctrl+a", "select all")),
			key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "deselect all")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		}
	}
	return selectListModel{Model: l}
}

func (m selectListModel) Init() tea.Cmd { return nil }

func (m selectListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Model.SetSize(msg.Width, listHeight)
		m.ready = true
		return m, nil

	case tea.KeyPressMsg:
		if m.Model.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "esc":
			if m.Model.FilterState() == list.FilterApplied {
				m.Model.ResetFilter()
				return m, nil
			}
			m.goBack = true
			return m, tea.Quit

		case "enter":
			m.done = true
			return m, tea.Quit

		case "space":
			if s, ok := m.Model.SelectedItem().(selectable); ok {
				cmd := m.Model.SetItem(m.Model.Index(), s.withSelected(!s.isSelected()))
				return m, cmd
			}
			return m, nil

		case "ctrl+a":
			var cmds []tea.Cmd
			for idx, listItem := range m.Model.Items() {
				if s, ok := listItem.(selectable); ok && !s.isSelected() {
					cmds = append(cmds, m.Model.SetItem(idx, s.withSelected(true)))
				}
			}
			return m, tea.Batch(cmds...)

		case "ctrl+d":
			var cmds []tea.Cmd
			for idx, listItem := range m.Model.Items() {
				if s, ok := listItem.(selectable); ok && s.isSelected() {
					cmds = append(cmds, m.Model.SetItem(idx, s.withSelected(false)))
				}
			}
			return m, tea.Batch(cmds...)
		}
	}

	var cmd tea.Cmd
	m.Model, cmd = m.Model.Update(msg)
	return m, cmd
}

// View renders the list with the alt screen enabled. Callers that need a
// custom view (e.g. context-aware help text) should render directly via
// the promoted list.Model.View() rather than using this method.
func (m selectListModel) View() tea.View {
	if !m.ready {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}
	if m.quitting {
		return tea.NewView(quitTextStyle.Render("Cancelling..."))
	}
	if m.goBack || m.done {
		return tea.NewView("")
	}
	v := tea.NewView("\n" + m.Model.View())
	v.AltScreen = true
	return v
}
