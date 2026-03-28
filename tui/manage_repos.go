package tui

import (
	"fmt"

	"github.com/jprincevevo/reap/config"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

func addRepo(cfg *config.Config, url string) bool {
	for _, r := range cfg.Repos {
		if r.URL == url {
			return false
		}
	}
	cfg.Repos = append(cfg.Repos, config.Repo{URL: url, Selected: true})
	return true
}

func removeRepoFromCfg(cfg *config.Config, url string) []config.Repo {
	var kept []config.Repo
	for _, r := range cfg.Repos {
		if r.URL != url {
			kept = append(kept, r)
		}
	}
	return kept
}

type manageRepoScreen int

const (
	mrScreenList manageRepoScreen = iota
	mrScreenAdd
	mrScreenRemove
)

type manageRepoModel struct {
	cfg    *config.Config
	screen manageRepoScreen
	list   list.Model
	prompt promptModel
	remove removeModel
	width  int
	ready  bool
	done   bool
}

func (m manageRepoModel) Init() tea.Cmd {
	return nil
}

func (m manageRepoModel) buildList() list.Model {
	var items []list.Item
	for _, repo := range m.cfg.Repos {
		items = append(items, item(repo.URL))
	}

	const defaultWidth = 20
	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Manage repositories"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "done")),
		}
	}

	if m.width > 0 {
		l.SetSize(m.width, listHeight)
	}

	return l
}

func (m manageRepoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
	}

	switch m.screen {
	case mrScreenList:
		return m.updateList(msg)
	case mrScreenAdd:
		return m.updateAdd(msg)
	case mrScreenRemove:
		return m.updateRemove(msg)
	}

	return m, nil
}

func (m manageRepoModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, listHeight)
		m.ready = true
		return m, nil

	case tea.KeyPressMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "ctrl+c", "q", "enter":
			m.done = true
			return m, tea.Quit

		case "a":
			m.screen = mrScreenAdd
			m.prompt = NewPromptModel("Add repository", "https://github.com/owner/repo.git")
			cmd := m.prompt.Init()
			return m, cmd

		case "d":
			if len(m.cfg.Repos) == 0 {
				return m, nil
			}
			m.screen = mrScreenRemove
			m.remove = NewRemoveModel(m.cfg)
			if m.width > 0 {
				m.remove.list.SetSize(m.width, listHeight)
				m.remove.ready = true
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m manageRepoModel) updateAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
	newPrompt, cmd := m.prompt.Update(msg)
	m.prompt = newPrompt.(promptModel)

	if m.prompt.quitting {
		url := m.prompt.input.Value()
		if url != "" {
			addRepo(m.cfg, url)
			if err := config.Save(m.cfg); err != nil {
				fmt.Println("Error saving config:", err)
			}
		}
		m.screen = mrScreenList
		m.list = m.buildList()
		return m, nil
	}

	return m, cmd
}

func (m manageRepoModel) updateRemove(msg tea.Msg) (tea.Model, tea.Cmd) {
	newRemove, cmd := m.remove.Update(msg)
	m.remove = newRemove.(removeModel)

	if m.remove.choice != "" {
		m.cfg.Repos = removeRepoFromCfg(m.cfg, m.remove.choice)
		if err := config.Save(m.cfg); err != nil {
			fmt.Println("Error saving config:", err)
		}
		m.screen = mrScreenList
		m.list = m.buildList()
		return m, nil
	}

	if m.remove.quitting {
		m.screen = mrScreenList
		m.list = m.buildList()
		return m, nil
	}

	return m, cmd
}

func (m manageRepoModel) View() tea.View {
	switch m.screen {
	case mrScreenAdd:
		return m.prompt.View()
	case mrScreenRemove:
		return m.remove.View()
	}

	if !m.ready {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}
	if m.done {
		return tea.NewView("")
	}
	v := tea.NewView("\n" + m.list.View())
	v.AltScreen = true
	return v
}

func InitialManageReposModel(cfg *config.Config) error {
	m := manageRepoModel{
		cfg: cfg,
	}
	m.list = m.buildList()

	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}
