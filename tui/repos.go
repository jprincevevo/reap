package tui

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jprincevevo/reap/config"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ErrGoBack is returned by InitialRepoModel when the user presses escape to
// go back to the previous screen.
var ErrGoBack = errors.New("go back")

var (
	checkedBadgeStyle   = lipgloss.NewStyle().Foreground(colorSuccess)
	uncheckedBadgeStyle = lipgloss.NewStyle().Foreground(colorDim)
	existsBadgeStyle    = lipgloss.NewStyle().Foreground(colorMuted)
	urlHostStyle        = lipgloss.NewStyle().Foreground(colorDim)
)

// splitURL splits a repository URL into a dim host prefix and the owner/repo
// path. Supports SSH (git@host:owner/repo.git) and HTTPS/HTTP.
func splitURL(rawURL string) (host, path string) {
	if strings.HasPrefix(rawURL, "git@") {
		if idx := strings.Index(rawURL, ":"); idx != -1 {
			return rawURL[:idx+1], strings.TrimSuffix(rawURL[idx+1:], ".git")
		}
	}
	for _, scheme := range []string{"https://", "http://"} {
		if strings.HasPrefix(rawURL, scheme) {
			rest := rawURL[len(scheme):]
			if idx := strings.Index(rest, "/"); idx != -1 {
				return scheme + rest[:idx+1], strings.TrimSuffix(rest[idx+1:], ".git")
			}
		}
	}
	return "", rawURL
}

type repoItem struct {
	url      string
	selected bool
	exists   bool
}

func (i repoItem) FilterValue() string { return i.url }
func (i repoItem) Title() string       { return i.url }
func (i repoItem) Description() string {
	if i.selected {
		return "[x]"
	}
	return "[ ]"
}

func (i repoItem) isSelected() bool              { return i.selected }
func (i repoItem) withSelected(s bool) list.Item { i.selected = s; return i }

type repoDelegate struct{}

func (d repoDelegate) Height() int                             { return 1 }
func (d repoDelegate) Spacing() int                            { return 0 }
func (d repoDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d repoDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(repoItem)
	if !ok {
		return
	}

	var badge string
	if i.exists && !i.selected {
		badge = existsBadgeStyle.Render("[~]")
	} else {
		badge = selectBadge(i.selected)
	}

	host, path := splitURL(i.url)
	urlStr := urlHostStyle.Render(host) + path

	var line string
	if index == m.Index() {
		line = "  " + cursorStyle.Render(">") + " " + badge + " " + urlStr
	} else {
		line = "    " + badge + " " + urlStr
	}
	fmt.Fprint(w, line)
}

type repoModel struct {
	list     selectListModel
	quitting bool
	goBack   bool
	ready    bool
}

func (m repoModel) Init() tea.Cmd {
	return nil
}

func (m repoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Set ready on WindowSizeMsg before delegating, so the flag is never
	// inadvertently reset by syncing from the inner list (which starts with
	// ready=false and may not have received a WindowSizeMsg if the parent
	// marked repoModel ready externally).
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		m.ready = true
	}
	newList, cmd := m.list.Update(msg)
	m.list = newList.(selectListModel)
	m.quitting = m.list.quitting
	m.goBack = m.list.goBack
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
	enterHelp := "confirm"
	if m.list.FilterState() == list.FilterApplied {
		enterHelp = "clone visible"
	}
	m.list.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle")),
			key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("ctrl+a", "select all")),
			key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "deselect all")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", enterHelp)),
			key.NewBinding(key.WithKeys("~"), key.WithHelp("~", "already in dir")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		}
	}
	v := tea.NewView("\n" + m.list.Model.View())
	v.AltScreen = true
	return v
}

func repoExists(url string) bool {
	repoName := strings.TrimSuffix(filepath.Base(url), ".git")
	_, err := os.Stat(repoName)
	return err == nil
}

func NewRepoModel(cfg *config.Config, group string) repoModel {
	var items []list.Item
	for _, repo := range cfg.Repos {
		if group == showAllRepos {
			exists := repoExists(repo.URL)
			sel := repo.Selected
			if exists {
				sel = false
			}
			items = append(items, repoItem{url: repo.URL, selected: sel, exists: exists})
		} else {
			for _, g := range repo.Groups {
				if g.Name == group {
					exists := repoExists(repo.URL)
					sel := g.Selected
					if exists {
						sel = false
					}
					items = append(items, repoItem{url: repo.URL, selected: sel, exists: exists})
				}
			}
		}
	}

	l := newSelectList(items, repoDelegate{}, "Select repositories to clone")

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
		for _, item := range m.list.VisibleItems() {
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
