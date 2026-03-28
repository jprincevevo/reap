package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jprincevevo/reap/config"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	listHeight       = 14
	listDefaultWidth = 20

	// showAllRepos is the sentinel group name that means "no group filter —
	// show every configured repository". It appears as the first item in the
	// group selection list and is matched in NewRepoModel to decide which
	// repos to show.
	showAllRepos = "Show all repos"
)

// selectBadge returns the styled "[✓]" or "[ ]" badge used by all selectable
// list delegates and toggle views. Centralising the rendering here ensures a
// consistent look across every screen.
func selectBadge(selected bool) string {
	if selected {
		return checkedBadgeStyle.Render("[✓]")
	}
	return uncheckedBadgeStyle.Render("[ ]")
}

// logSaveErr prints a config save error to stdout. config.Save failures cannot
// be surfaced through the TUI event loop, so stderr/stdout is the fallback.
func logSaveErr(err error) {
	if err != nil {
		fmt.Println("Error saving config:", err)
	}
}

// newList creates a list.Model with the shared visual style applied. Callers
// set AdditionalShortHelpKeys and call SetSize after construction.
func newList(items []list.Item, delegate list.ItemDelegate, title string) list.Model {
	l := list.New(items, delegate, listDefaultWidth, listHeight)
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	l.Help.Styles = dimHelpStyles
	return l
}

const bannerText = `
  ██████╗ ███████╗ █████╗ ██████╗
  ██╔══██╗██╔════╝██╔══██╗██╔══██╗
  ██████╔╝█████╗  ███████║██████╔╝
  ██╔══██╗██╔══╝  ██╔══██║██╔═══╝
  ██║  ██║███████╗██║  ██║██║
  ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═╝`

// Color palette — shared across all TUI screens.
// Uses LightDark to adapt to dark/light terminal backgrounds at startup.
var lightDark = lipgloss.LightDark(lipgloss.HasDarkBackground(os.Stdin, os.Stdout))

var (
	colorAccent  = lightDark(lipgloss.Color("#FF9F1C"), lipgloss.Color("#FF9F1C")) // amber
	colorSuccess = lightDark(lipgloss.Color("#2EC4B6"), lipgloss.Color("#2EC4B6")) // teal
	colorError   = lightDark(lipgloss.Color("#b91c1c"), lipgloss.Color("#ef4444")) // red (semantic override)
	colorDim     = lightDark(lipgloss.Color("#9ca3af"), lipgloss.Color("#c9cdd4")) // gray
	colorMuted   = lightDark(lipgloss.Color("#a05c00"), lipgloss.Color("#FFBF69")) // peach / dark amber
	colorPurple  = lightDark(lipgloss.Color("#874BFD"), lipgloss.Color("#874BFD")) // purple (confirm dialog border)
)

var (
	titleStyle        = lipgloss.NewStyle().Bold(true).Foreground(colorAccent).MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4).Foreground(colorDim)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(colorAccent)
	cursorStyle       = lipgloss.NewStyle().Foreground(colorAccent)
	paginationStyle   = list.DefaultStyles(false).PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles(false).HelpStyle.PaddingLeft(4).PaddingBottom(1).Foreground(colorAccent)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

// dimHelpStyles overrides the bubbles help.Model's per-token colors so that
// all key/description/separator text uses colorDim rather than the library's
// own near-invisible dark defaults.
var dimHelpStyles = help.Styles{
	ShortKey:       lipgloss.NewStyle().Foreground(colorMuted),
	ShortDesc:      lipgloss.NewStyle().Foreground(colorDim),
	ShortSeparator: lipgloss.NewStyle().Foreground(colorDim),
	Ellipsis:       lipgloss.NewStyle().Foreground(colorDim),
	FullKey:        lipgloss.NewStyle().Foreground(colorMuted),
	FullDesc:       lipgloss.NewStyle().Foreground(colorDim),
	FullSeparator:  lipgloss.NewStyle().Foreground(colorDim),
}

type item string

func (i item) FilterValue() string { return string(i) }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
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

type groupModel struct {
	list      navListModel
	choice    string
	quitting  bool
	ready     bool
	repoCount int // total repos in config, used to show onboarding when zero
	width     int
}

func (m groupModel) Init() tea.Cmd {
	return nil
}

func (m groupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.ready = true
	}

	var event navListEvent
	var cmd tea.Cmd
	m.list, event, cmd = m.list.Update(msg)

	switch event {
	case navListEventQuit, navListEventGoBack:
		m.quitting = true
		return m, tea.Quit
	case navListEventEnter:
		if i, ok := m.list.SelectedItem().(item); ok {
			m.choice = string(i)
		}
		return m, tea.Quit
	}

	return m, cmd
}

func (m groupModel) View() tea.View {
	if !m.ready {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}
	if m.choice != "" {
		return tea.NewView(quitTextStyle.Render(fmt.Sprintf("Selected group: %s", m.choice)))
	}
	if m.quitting {
		return tea.NewView(quitTextStyle.Render("Cancelling..."))
	}
	// Onboarding: no repos configured yet.
	if m.repoCount == 0 {
		bannerStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
		dimStyle := lipgloss.NewStyle().Foreground(colorDim).MarginLeft(2)
		cmdStyle := lipgloss.NewStyle().Foreground(colorMuted).MarginLeft(4)
		mutedInline := lipgloss.NewStyle().Foreground(colorMuted)
		keyStyle := lipgloss.NewStyle().Foreground(colorAccent)

		content := bannerStyle.Render(bannerText) + "\n\n" +
			dimStyle.Render("Batch-clone Git repos from a YAML config file.") + "\n\n" +
			dimStyle.Render("Get started:") + "\n" +
			cmdStyle.Render("Press ") + keyStyle.Render("r") + mutedInline.Render(" to open repo management") + "\n" +
			cmdStyle.Render("Or run:  reap repo add <url>") + "\n\n" +
			dimStyle.Render("GitHub:") + "\n" +
			cmdStyle.Render("github.com/jprincevevo/reap") + "\n\n" +
			dimStyle.Render("Press ") + keyStyle.Render("q") + lipgloss.NewStyle().Foreground(colorDim).Render(" to quit.")
		v := tea.NewView(content)
		v.AltScreen = true
		return v
	}
	v := tea.NewView("\n" + m.list.View())
	v.AltScreen = true
	return v
}

func NewGroupModel(cfg *config.Config) groupModel {

	groups := []list.Item{item(showAllRepos)}

	seenGroups := make(map[string]bool)
	for _, repo := range cfg.Repos {
		for _, group := range repo.Groups {
			if !seenGroups[group.Name] {
				seenGroups[group.Name] = true
				groups = append(groups, item(group.Name))
			}
		}
	}

	l := newNavList(groups, itemDelegate{}, "Select a group")
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "groups")),
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "repos")),
			key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "settings")),
		}
	}

	return groupModel{list: l, repoCount: len(cfg.Repos)}
}

func InitialGroupModel(cfg *config.Config) (string, error) {
	m := NewGroupModel(cfg)

	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	if m, ok := finalModel.(groupModel); ok && m.choice != "" {
		return m.choice, nil
	}

	return "", fmt.Errorf("no group selected")
}
