package tui

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jprincevevo/reap/config"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

// ErrGoBack is returned by InitialRepoModel when the user presses escape to
// go back to the previous screen.
var ErrGoBack = errors.New("go back")

type repoItem struct {
	url      string
	selected bool
}

func (i repoItem) FilterValue() string { return i.url }
func (i repoItem) Title() string       { return i.url }
func (i repoItem) Description() string {
	if i.selected {
		return "[x]"
	}
	return "[ ]"
}

type repoDelegate struct{}

func (d repoDelegate) Height() int                             { return 1 }
func (d repoDelegate) Spacing() int                            { return 0 }
func (d repoDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d repoDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(repoItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s %s", i.Description(), i.Title())

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type repoModel struct {
	list     list.Model
	quitting bool
	goBack   bool
	ready    bool
}

func (m repoModel) Init() tea.Cmd {
	return nil
}

func (m repoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

		case "esc":
			m.goBack = true
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

func (m repoModel) View() tea.View {
	if !m.ready {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}
	if m.quitting {
		return tea.NewView(quitTextStyle.Render("Cancelling..."))
	}
	if m.goBack {
		return tea.NewView("")
	}
	v := tea.NewView("\n" + m.list.View())
	v.AltScreen = true
	return v
}

func NewRepoModel(cfg *config.Config, group string) repoModel {
	var items []list.Item
	for _, repo := range cfg.Repos {
		if group == "Show All" {
			items = append(items, repoItem{url: repo.URL, selected: repo.Selected})
		} else {
			for _, g := range repo.Groups {
				if g.Name == group {
					items = append(items, repoItem{url: repo.URL, selected: g.Selected})
				}
			}
		}
	}

	const defaultWidth = 20

	l := list.New(items, repoDelegate{}, defaultWidth, listHeight)
	l.Title = "Select repositories to clone"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("space"),
				key.WithHelp("space", "toggle"),
			),
			key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "confirm"),
			),
			key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "back"),
			),
			key.NewBinding(
				key.WithKeys("ctrl+c", "q"),
				key.WithHelp("ctrl+c/q", "quit"),
			),
		}
	}

	return repoModel{list: l}
}

func InitialRepoModel(cfg *config.Config, group string) ([]string, error) {
	m := NewRepoModel(cfg, group)

	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	var selected []string
	if m, ok := finalModel.(repoModel); ok {
		if m.goBack {
			return nil, ErrGoBack
		}
		if m.quitting {
			return nil, fmt.Errorf("aborted")
		}
		for _, item := range m.list.Items() {
			if i, ok := item.(repoItem); ok && i.selected {
				selected = append(selected, i.url)
			}
		}
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("no repositories selected")
	}

	return selected, nil
}
