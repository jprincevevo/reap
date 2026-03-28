package tui

import (
	"fmt"

	"github.com/jprincevevo/reap/config"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

type manageRepoScreen int

const (
	mrScreenList manageRepoScreen = iota
	mrScreenAdd
	mrScreenRemove
	mrScreenDetail
	mrScreenAddToGroup
	mrScreenConfirmRemoveGroup
)

type manageRepoModel struct {
	cfg          *config.Config
	screen       manageRepoScreen
	list         navListModel
	prompt       promptModel
	remove       removeModel
	width        int
	ready        bool
	goBack       bool
	selectedRepo string
	detailList   navListModel
	groupList    navListModel
	pendingGroup string
}

func (m manageRepoModel) Init() tea.Cmd {
	return nil
}

func (m manageRepoModel) buildList() navListModel {
	var items []list.Item
	for _, repo := range m.cfg.Repos {
		items = append(items, item(repo.URL))
	}

	l := newNavList(items, itemDelegate{}, "Manage repositories")
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		}
	}
	if m.width > 0 {
		l.SetSize(m.width, listHeight)
	}
	return l
}

func (m manageRepoModel) buildDetailList() navListModel {
	var items []list.Item
	for _, repo := range m.cfg.Repos {
		if repo.URL == m.selectedRepo {
			for _, g := range repo.Groups {
				items = append(items, item(g.Name))
			}
			break
		}
	}

	l := newNavList(items, itemDelegate{}, m.selectedRepo)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add to group")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "remove from group")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		}
	}
	if m.width > 0 {
		l.SetSize(m.width, listHeight)
	}
	return l
}

// buildGroupList returns all groups the selected repo does NOT already belong to.
func (m manageRepoModel) buildGroupList() navListModel {
	repoGroups := make(map[string]bool)
	for _, repo := range m.cfg.Repos {
		if repo.URL == m.selectedRepo {
			for _, g := range repo.Groups {
				repoGroups[g.Name] = true
			}
			break
		}
	}

	var items []list.Item
	seen := make(map[string]bool)
	for _, repo := range m.cfg.Repos {
		for _, g := range repo.Groups {
			if !seen[g.Name] && !repoGroups[g.Name] {
				seen[g.Name] = true
				items = append(items, item(g.Name))
			}
		}
	}

	l := newNavList(items, itemDelegate{}, "Add to group")
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
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
	case mrScreenDetail:
		return m.updateDetail(msg)
	case mrScreenAddToGroup:
		return m.updateAddToGroup(msg)
	case mrScreenConfirmRemoveGroup:
		return m.updateConfirmRemoveGroup(msg)
	}

	return m, nil
}

func (m manageRepoModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		m.ready = true
	}

	// Domain-specific keys handled before passing to navListModel.
	if kp, ok := msg.(tea.KeyPressMsg); ok && m.list.FilterState() != list.Filtering {
		switch kp.String() {
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

	var event navListEvent
	var cmd tea.Cmd
	m.list, event, cmd = m.list.Update(msg)

	switch event {
	case navListEventQuit:
		return m, cmd
	case navListEventGoBack:
		m.goBack = true
		return m, tea.Quit
	case navListEventEnter:
		sel, ok := m.list.SelectedItem().(item)
		if !ok {
			return m, nil
		}
		m.selectedRepo = string(sel)
		m.screen = mrScreenDetail
		m.detailList = m.buildDetailList()
		return m, nil
	}

	return m, cmd
}

func (m manageRepoModel) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Domain-specific keys handled before passing to navListModel.
	if kp, ok := msg.(tea.KeyPressMsg); ok && m.detailList.FilterState() != list.Filtering {
		switch kp.String() {
		case "a":
			m.screen = mrScreenAddToGroup
			m.groupList = m.buildGroupList()
			return m, nil

		case "d":
			sel, ok := m.detailList.SelectedItem().(item)
			if !ok {
				return m, nil
			}
			m.pendingGroup = string(sel)
			m.screen = mrScreenConfirmRemoveGroup
			m.prompt = NewPromptModel(
				fmt.Sprintf("Remove from group %q? Type 'yes' to confirm", m.pendingGroup),
				"yes",
			)
			cmd := m.prompt.Init()
			return m, cmd
		}
	}

	var event navListEvent
	var cmd tea.Cmd
	m.detailList, event, cmd = m.detailList.Update(msg)

	switch event {
	case navListEventQuit:
		return m, cmd
	case navListEventGoBack:
		m.screen = mrScreenList
		m.list = m.buildList()
		return m, nil
	}

	return m, cmd
}

func (m manageRepoModel) updateAddToGroup(msg tea.Msg) (tea.Model, tea.Cmd) {
	var event navListEvent
	var cmd tea.Cmd
	m.groupList, event, cmd = m.groupList.Update(msg)

	switch event {
	case navListEventQuit:
		return m, cmd

	case navListEventGoBack:
		m.screen = mrScreenDetail
		m.detailList = m.buildDetailList()
		return m, nil

	case navListEventEnter:
		sel, ok := m.groupList.SelectedItem().(item)
		if !ok {
			m.screen = mrScreenDetail
			m.detailList = m.buildDetailList()
			return m, nil
		}
		groupName := string(sel)
		for i, repo := range m.cfg.Repos {
			if repo.URL == m.selectedRepo {
				alreadyHas := false
				for _, g := range repo.Groups {
					if g.Name == groupName {
						alreadyHas = true
						break
					}
				}
				if !alreadyHas {
					m.cfg.Repos[i].Groups = append(m.cfg.Repos[i].Groups, config.Group{
						Name:     groupName,
						Selected: true,
					})
				}
				break
			}
		}
		logSaveErr(config.Save(m.cfg))
		m.screen = mrScreenDetail
		m.detailList = m.buildDetailList()
		return m, nil
	}

	return m, cmd
}

func (m manageRepoModel) updateAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
	newPrompt, cmd := m.prompt.Update(msg)
	m.prompt = newPrompt.(promptModel)

	if m.prompt.quitting {
		url := m.prompt.input.Value()
		if url != "" {
			m.cfg.AddRepo(url)
			logSaveErr(config.Save(m.cfg))
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
		m.cfg.RemoveRepo(m.remove.choice)
		logSaveErr(config.Save(m.cfg))
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

func (m manageRepoModel) updateConfirmRemoveGroup(msg tea.Msg) (tea.Model, tea.Cmd) {
	newPrompt, cmd := m.prompt.Update(msg)
	m.prompt = newPrompt.(promptModel)

	if m.prompt.quitting {
		if m.prompt.input.Value() == "yes" {
			for i, repo := range m.cfg.Repos {
				if repo.URL == m.selectedRepo {
					var kept []config.Group
					for _, g := range repo.Groups {
						if g.Name != m.pendingGroup {
							kept = append(kept, g)
						}
					}
					m.cfg.Repos[i].Groups = kept
					break
				}
			}
			logSaveErr(config.Save(m.cfg))
		}
		m.screen = mrScreenDetail
		m.detailList = m.buildDetailList()
		return m, nil
	}

	return m, cmd
}

func (m manageRepoModel) View() tea.View {
	switch m.screen {
	case mrScreenAdd, mrScreenConfirmRemoveGroup:
		return m.prompt.View()
	case mrScreenRemove:
		return m.remove.View()
	case mrScreenDetail:
		if !m.ready {
			v := tea.NewView("")
			v.AltScreen = true
			return v
		}
		v := tea.NewView("\n" + m.detailList.View())
		v.AltScreen = true
		return v
	case mrScreenAddToGroup:
		if !m.ready {
			v := tea.NewView("")
			v.AltScreen = true
			return v
		}
		v := tea.NewView("\n" + m.groupList.View())
		v.AltScreen = true
		return v
	}

	if !m.ready {
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}
	if m.goBack {
		return tea.NewView("")
	}
	// Onboarding: no repos configured yet.
	if len(m.list.Items()) == 0 {
		content := "\n" + titleStyle.Render("Manage repositories") + "\n\n" +
			"  No repositories configured yet.\n\n" +
			"  Press a to add your first repository, or run:\n" +
			"    reap repo add <url>\n\n" +
			"  Press q to go back.\n"
		v := tea.NewView(content)
		v.AltScreen = true
		return v
	}
	v := tea.NewView("\n" + m.list.View())
	v.AltScreen = true
	return v
}

// newManageRepoModel constructs a manageRepoModel ready for embedding inside
// a parent program (e.g. appModel). The list is built immediately; sizing
// happens when the parent forwards a WindowSizeMsg or sets dimensions directly.
func newManageRepoModel(cfg *config.Config) manageRepoModel {
	m := manageRepoModel{cfg: cfg}
	m.list = m.buildList()
	return m
}

func InitialManageReposModel(cfg *config.Config) error {
	m := newManageRepoModel(cfg)
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}
