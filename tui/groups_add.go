package tui

import (
	"fmt"
	"io"
	"os"
	"reap/config"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type groupAddItem repoItem

func (i groupAddItem) FilterValue() string { return i.url }
func (i groupAddItem) Title() string       { return i.url }
func (i groupAddItem) Description() string {
	if i.selected {
		return "[x]"
	}
	return "[ ]"
}

type groupAddDelegate struct{}

func (d groupAddDelegate) Height() int                             { return 1 }
func (d groupAddDelegate) Spacing() int                            { return 0 }
func (d groupAddDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d groupAddDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(groupAddItem)
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

type groupAddModel struct {
	list     list.Model
	quitting bool
}

func (m groupAddModel) Init() tea.Cmd {
	return nil
}

func (m groupAddModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			return m, tea.Quit

		case " ":
			if i, ok := m.list.SelectedItem().(groupAddItem); ok {
				i.selected = !i.selected
				m.list.SetItem(m.list.Index(), i)
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m groupAddModel) View() string {
	if m.quitting {
		return quitTextStyle.Render("Cancelling...")
	}
	return "\n" + m.list.View()
}

func NewGroupAddModel(cfg *config.Config) groupAddModel {
	var items []list.Item
	for _, repo := range cfg.Repos {
		items = append(items, groupAddItem{url: repo.URL, selected: false})
	}

	const defaultWidth = 20

	l := list.New(items, groupAddDelegate{}, defaultWidth, listHeight)
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
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	var selected []string
	if m, ok := finalModel.(groupAddModel); ok {
		for _, item := range m.list.Items() {
			if i, ok := item.(groupAddItem); ok && i.selected {
				selected = append(selected, i.url)
			}
		}
	}

	return selected, nil
}
