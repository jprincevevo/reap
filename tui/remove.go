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

type removeItem item

func (i removeItem) FilterValue() string { return "" }

type removeDelegate struct{}

func (d removeDelegate) Height() int                             { return 1 }
func (d removeDelegate) Spacing() int                            { return 0 }
func (d removeDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d removeDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(removeItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type removeModel struct {
	list     list.Model
	choice   string
	quitting bool
}

func (m removeModel) Init() tea.Cmd {
	return nil
}

func (m removeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			i, ok := m.list.SelectedItem().(removeItem)
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

func (m removeModel) View() string {
	if m.choice != "" {
		return quitTextStyle.Render(fmt.Sprintf("Selected repository: %s", m.choice))
	}
	if m.quitting {
		return quitTextStyle.Render("Cancelling...")
	}
	return "\n" + m.list.View()
}

func NewRemoveModel(cfg *config.Config) removeModel {
	var items []list.Item
	for _, repo := range cfg.Repos {
		items = append(items, removeItem(repo.URL))
	}

	const defaultWidth = 20

	l := list.New(items, removeDelegate{}, defaultWidth, listHeight)
	l.Title = "Select a repository to remove"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	return removeModel{list: l}
}

func InitialRemoveModel(cfg *config.Config) (string, error) {
	m := NewRemoveModel(cfg)

	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	if m, ok := finalModel.(removeModel); ok && m.choice != "" {
		return m.choice, nil
	}

	return "", fmt.Errorf("no repository selected")
}
