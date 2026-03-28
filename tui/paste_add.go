package tui

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/jprincevevo/reap/config"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// sanitizeGitHubURL accepts a raw string (HTTPS or SSH GitHub URL, possibly
// with extra path segments like /tree/main) and returns the canonical
// https://github.com/owner/repo.git form. Returns ("", false) if the input
// is not a recognisable GitHub repo URL.
func sanitizeGitHubURL(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)

	// SSH: git@github.com:owner/repo[.git]
	if strings.HasPrefix(raw, "git@github.com:") {
		path := strings.TrimPrefix(raw, "git@github.com:")
		path = strings.TrimSuffix(path, ".git")
		parts := strings.SplitN(path, "/", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return fmt.Sprintf("https://github.com/%s/%s.git", parts[0], parts[1]), true
		}
		return "", false
	}

	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host != "github.com" {
		return "", false
	}

	// Require at least /owner/repo — strip any deeper path (e.g. /tree/main).
	parts := strings.SplitN(strings.Trim(u.Path, "/"), "/", 3)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}

	owner := parts[0]
	repo := strings.TrimSuffix(parts[1], ".git")
	if repo == "" {
		return "", false
	}

	return fmt.Sprintf("https://github.com/%s/%s.git", owner, repo), true
}

// ---- group multi-select list ------------------------------------------------

type groupSelectItem struct {
	name     string
	selected bool
}

func (g groupSelectItem) FilterValue() string           { return g.name }
func (g groupSelectItem) isSelected() bool              { return g.selected }
func (g groupSelectItem) withSelected(s bool) list.Item { g.selected = s; return g }

type groupSelectDelegate struct{}

func (d groupSelectDelegate) Height() int                             { return 1 }
func (d groupSelectDelegate) Spacing() int                            { return 0 }
func (d groupSelectDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d groupSelectDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	gi, ok := listItem.(groupSelectItem)
	if !ok {
		return
	}

	str := selectBadge(gi.selected) + " " + gi.name

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

// ---- pasteAddModel ----------------------------------------------------------

type pasteAddScreen int

const (
	pasteAddScreenConfirm pasteAddScreen = iota
	pasteAddScreenGroups
)

type pasteAddModel struct {
	cfg       *config.Config
	screen    pasteAddScreen
	url       string
	groupList selectListModel
	hasGroups bool // whether any groups exist (skip group screen if false)
	done      bool
	cancelled bool // esc — soft cancel, appModel transitions back; no tea.Quit propagation
	width     int
	ready     bool
}

func newPasteAddModel(cfg *config.Config, sanitizedURL, preselected string) pasteAddModel {
	seen := make(map[string]bool)
	var items []list.Item
	for _, repo := range cfg.Repos {
		for _, g := range repo.Groups {
			if !seen[g.Name] {
				seen[g.Name] = true
				items = append(items, groupSelectItem{
					name:     g.Name,
					selected: g.Name == preselected,
				})
			}
		}
	}

	l := newSelectList(items, groupSelectDelegate{}, "Add to groups (optional)")
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle")),
			key.NewBinding(key.WithKeys("ctrl+a"), key.WithHelp("ctrl+a", "select all")),
			key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "deselect all")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		}
	}

	return pasteAddModel{
		cfg:       cfg,
		url:       sanitizedURL,
		groupList: l,
		hasGroups: len(items) > 0,
		ready:     true,
	}
}

func (m pasteAddModel) Init() tea.Cmd { return nil }

func (m pasteAddModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		// Route through selectListModel.Update so it sizes the list and sets ready.
		newGL, _ := m.groupList.Update(wsm)
		m.groupList = newGL.(selectListModel)
		return m, nil
	}

	switch m.screen {
	case pasteAddScreenConfirm:
		return m.updateConfirm(msg)
	case pasteAddScreenGroups:
		return m.updateGroups(msg)
	}
	return m, nil
}

func (m pasteAddModel) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch kp.String() {
	case "ctrl+c", "q":
		return m, tea.Quit // hard quit — propagates out of appModel

	case "esc":
		m.cancelled = true
		return m, tea.Quit // soft cancel — appModel intercepts via cancelled flag

	case "enter":
		if !m.hasGroups {
			// No groups exist — save immediately and finish.
			m.cfg.AddRepo(m.url)
			logSaveErr(config.Save(m.cfg))
			m.done = true
			return m, tea.Quit
		}
		m.screen = pasteAddScreenGroups
		return m, nil
	}

	return m, nil
}

func (m pasteAddModel) updateGroups(msg tea.Msg) (tea.Model, tea.Cmd) {
	newGL, cmd := m.groupList.Update(msg)
	m.groupList = newGL.(selectListModel)

	if m.groupList.quitting {
		return m, cmd // hard quit — propagate tea.Quit
	}

	if m.groupList.goBack {
		// Esc from the groups screen — go back to confirm without quitting.
		// Reset the flag so the list works correctly if the user re-enters.
		m.groupList.goBack = false
		m.screen = pasteAddScreenConfirm
		return m, nil
	}

	if m.groupList.done {
		m.groupList.done = false
		m.cfg.AddRepo(m.url)
		for _, listItem := range m.groupList.Items() {
			gi, ok := listItem.(groupSelectItem)
			if !ok || !gi.selected {
				continue
			}
			for i, repo := range m.cfg.Repos {
				if repo.URL == m.url {
					m.cfg.Repos[i].Groups = append(m.cfg.Repos[i].Groups, config.Group{
						Name:     gi.name,
						Selected: true,
					})
					break
				}
			}
		}
		logSaveErr(config.Save(m.cfg))
		m.done = true
		return m, tea.Quit
	}

	return m, cmd
}

func (m pasteAddModel) View() tea.View {
	switch m.screen {
	case pasteAddScreenConfirm:
		dimStyle := lipgloss.NewStyle().Foreground(colorDim).MarginLeft(2)
		urlStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).MarginLeft(4)

		hint := "enter  confirm  •  esc  cancel"
		if m.hasGroups {
			hint = "enter  choose groups  •  esc  cancel"
		}

		content := "\n" +
			dimStyle.Render("Add repository?") + "\n\n" +
			urlStyle.Render(m.url) + "\n\n" +
			dimStyle.Render(hint)

		v := tea.NewView(content)
		v.AltScreen = true
		return v

	case pasteAddScreenGroups:
		v := tea.NewView("\n" + m.groupList.Model.View())
		v.AltScreen = true
		return v
	}

	return tea.NewView("")
}
