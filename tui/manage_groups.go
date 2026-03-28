package tui

import (
	"fmt"

	"github.com/jprincevevo/reap/config"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)


type manageGroupScreen int

const (
	mgScreenList manageGroupScreen = iota
	mgScreenAdd
	mgScreenAddRepos
	mgScreenRename
	mgScreenConfirmDelete
	mgScreenDetail
	mgScreenConfirmRemoveRepo
	mgScreenAddRepoToGroup
)

type manageGroupModel struct {
	cfg           *config.Config
	screen        manageGroupScreen
	list          list.Model
	detailList    list.Model
	repoList      list.Model
	prompt        promptModel
	groupAdd      groupAddModel
	width         int
	ready         bool
	goBack        bool
	pendingGroup  string
	selectedGroup string
	selectedRepo  string
}

func (m manageGroupModel) Init() tea.Cmd {
	return nil
}

func (m manageGroupModel) buildList() list.Model {
	var items []list.Item
	seen := make(map[string]bool)
	for _, repo := range m.cfg.Repos {
		for _, g := range repo.Groups {
			if !seen[g.Name] {
				seen[g.Name] = true
				items = append(items, item(g.Name))
			}
		}
	}

	l := newList(items, itemDelegate{}, "Manage groups")
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		}
	}
	if m.width > 0 {
		l.SetSize(m.width, listHeight)
	}
	return l
}

func (m manageGroupModel) buildDetailList() list.Model {
	var items []list.Item
	for _, repo := range m.cfg.Repos {
		for _, g := range repo.Groups {
			if g.Name == m.selectedGroup {
				items = append(items, item(repo.URL))
				break
			}
		}
	}

	l := newList(items, itemDelegate{}, "Group: "+m.selectedGroup)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add repo")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "remove repo")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		}
	}
	if m.width > 0 {
		l.SetSize(m.width, listHeight)
	}
	return l
}

// buildRepoAddList returns all repos that are NOT already in m.selectedGroup.
func (m manageGroupModel) buildRepoAddList() list.Model {
	var items []list.Item
	for _, repo := range m.cfg.Repos {
		inGroup := false
		for _, g := range repo.Groups {
			if g.Name == m.selectedGroup {
				inGroup = true
				break
			}
		}
		if !inGroup {
			items = append(items, item(repo.URL))
		}
	}

	l := newList(items, itemDelegate{}, "Add repo to: "+m.selectedGroup)
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

func (m manageGroupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
	}

	switch m.screen {
	case mgScreenList:
		return m.updateList(msg)
	case mgScreenAdd:
		return m.updateAdd(msg)
	case mgScreenAddRepos:
		return m.updateAddRepos(msg)
	case mgScreenRename:
		return m.updateRename(msg)
	case mgScreenConfirmDelete:
		return m.updateConfirmDelete(msg)
	case mgScreenDetail:
		return m.updateDetail(msg)
	case mgScreenConfirmRemoveRepo:
		return m.updateConfirmRemoveRepo(msg)
	case mgScreenAddRepoToGroup:
		return m.updateAddRepoToGroup(msg)
	}

	return m, nil
}

func (m manageGroupModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			m.goBack = true
			return m, tea.Quit

		case "enter":
			sel, ok := m.list.SelectedItem().(item)
			if !ok {
				return m, nil
			}
			m.selectedGroup = string(sel)
			m.screen = mgScreenDetail
			m.detailList = m.buildDetailList()
			return m, nil

		case "a":
			m.screen = mgScreenAdd
			m.prompt = NewPromptModel("New group name", "my-group")
			cmd := m.prompt.Init()
			return m, cmd

		case "r":
			sel, ok := m.list.SelectedItem().(item)
			if !ok {
				return m, nil
			}
			m.selectedGroup = string(sel)
			m.screen = mgScreenRename
			m.prompt = NewPromptModelWithValue("Rename group", "new-name", m.selectedGroup)
			cmd := m.prompt.Init()
			return m, cmd

		case "d":
			sel, ok := m.list.SelectedItem().(item)
			if !ok {
				return m, nil
			}
			m.selectedGroup = string(sel)
			m.screen = mgScreenConfirmDelete
			m.prompt = NewPromptModel(
				fmt.Sprintf("Delete group %q? Type 'yes' to confirm", m.selectedGroup),
				"yes",
			)
			cmd := m.prompt.Init()
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m manageGroupModel) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.detailList.SetSize(msg.Width, listHeight)
		return m, nil

	case tea.KeyPressMsg:
		if m.detailList.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			m.screen = mgScreenList
			m.list = m.buildList()
			return m, nil

		case "a":
			m.screen = mgScreenAddRepoToGroup
			m.repoList = m.buildRepoAddList()
			return m, nil

		case "d":
			sel, ok := m.detailList.SelectedItem().(item)
			if !ok {
				return m, nil
			}
			m.selectedRepo = string(sel)
			m.screen = mgScreenConfirmRemoveRepo
			m.prompt = NewPromptModel(
				fmt.Sprintf("Remove %q from group %q? Type 'yes' to confirm", m.selectedRepo, m.selectedGroup),
				"yes",
			)
			cmd := m.prompt.Init()
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.detailList, cmd = m.detailList.Update(msg)
	return m, cmd
}

func (m manageGroupModel) updateAddRepoToGroup(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.repoList.SetSize(msg.Width, listHeight)
		return m, nil

	case tea.KeyPressMsg:
		if m.repoList.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			m.screen = mgScreenDetail
			m.detailList = m.buildDetailList()
			return m, nil

		case "enter":
			sel, ok := m.repoList.SelectedItem().(item)
			if !ok {
				m.screen = mgScreenDetail
				m.detailList = m.buildDetailList()
				return m, nil
			}
			repoURL := string(sel)
			for i, repo := range m.cfg.Repos {
				if repo.URL == repoURL {
					alreadyHas := false
					for _, g := range repo.Groups {
						if g.Name == m.selectedGroup {
							alreadyHas = true
							break
						}
					}
					if !alreadyHas {
						m.cfg.Repos[i].Groups = append(m.cfg.Repos[i].Groups, config.Group{
							Name:     m.selectedGroup,
							Selected: true,
						})
					}
					break
				}
			}
			logSaveErr(config.Save(m.cfg))
			m.screen = mgScreenDetail
			m.detailList = m.buildDetailList()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.repoList, cmd = m.repoList.Update(msg)
	return m, cmd
}

func (m manageGroupModel) updateAdd(msg tea.Msg) (tea.Model, tea.Cmd) {
	newPrompt, cmd := m.prompt.Update(msg)
	m.prompt = newPrompt.(promptModel)

	if m.prompt.quitting {
		name := m.prompt.input.Value()
		if name != "" {
			m.pendingGroup = name
			m.screen = mgScreenAddRepos
			m.groupAdd = NewGroupAddModel(m.cfg)
			if m.width > 0 {
				m.groupAdd.list.SetSize(m.width, listHeight)
				m.groupAdd.ready = true
			}
			return m, nil
		}
		m.screen = mgScreenList
		m.list = m.buildList()
		return m, nil
	}

	return m, cmd
}

func (m manageGroupModel) updateAddRepos(msg tea.Msg) (tea.Model, tea.Cmd) {
	newGA, cmd := m.groupAdd.Update(msg)
	m.groupAdd = newGA.(groupAddModel)

	if m.groupAdd.quitting {
		m.screen = mgScreenList
		m.list = m.buildList()
		return m, nil
	}

	finished := false
	if km, ok := msg.(tea.KeyPressMsg); ok && km.String() == "enter" {
		finished = true
	}

	if finished {
		var selectedURLs []string
		for _, listItem := range m.groupAdd.list.Items() {
			if ri, ok := listItem.(repoItem); ok && ri.selected {
				selectedURLs = append(selectedURLs, ri.url)
			}
		}
		if len(selectedURLs) > 0 {
			m.cfg.ApplyGroupToRepos(m.pendingGroup, selectedURLs)
			logSaveErr(config.Save(m.cfg))
		}
		m.screen = mgScreenList
		m.list = m.buildList()
		return m, nil
	}

	return m, cmd
}

func (m manageGroupModel) updateRename(msg tea.Msg) (tea.Model, tea.Cmd) {
	newPrompt, cmd := m.prompt.Update(msg)
	m.prompt = newPrompt.(promptModel)

	if m.prompt.quitting {
		newName := m.prompt.input.Value()
		if newName != "" && newName != m.selectedGroup {
			for i, repo := range m.cfg.Repos {
				for j, g := range repo.Groups {
					if g.Name == m.selectedGroup {
						m.cfg.Repos[i].Groups[j].Name = newName
					}
				}
			}
			logSaveErr(config.Save(m.cfg))
		}
		m.screen = mgScreenList
		m.list = m.buildList()
		return m, nil
	}

	return m, cmd
}

func (m manageGroupModel) updateConfirmDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	newPrompt, cmd := m.prompt.Update(msg)
	m.prompt = newPrompt.(promptModel)

	if m.prompt.quitting {
		if m.prompt.input.Value() == "yes" {
			m.cfg.RemoveGroupFromAllRepos(m.selectedGroup)
			logSaveErr(config.Save(m.cfg))
		}
		m.screen = mgScreenList
		m.list = m.buildList()
		return m, nil
	}

	return m, cmd
}

func (m manageGroupModel) updateConfirmRemoveRepo(msg tea.Msg) (tea.Model, tea.Cmd) {
	newPrompt, cmd := m.prompt.Update(msg)
	m.prompt = newPrompt.(promptModel)

	if m.prompt.quitting {
		if m.prompt.input.Value() == "yes" {
			for i, repo := range m.cfg.Repos {
				if repo.URL == m.selectedRepo {
					var kept []config.Group
					for _, g := range repo.Groups {
						if g.Name != m.selectedGroup {
							kept = append(kept, g)
						}
					}
					m.cfg.Repos[i].Groups = kept
					break
				}
			}
			logSaveErr(config.Save(m.cfg))
		}
		m.screen = mgScreenDetail
		m.detailList = m.buildDetailList()
		return m, nil
	}

	return m, cmd
}

func (m manageGroupModel) View() tea.View {
	switch m.screen {
	case mgScreenAdd, mgScreenRename, mgScreenConfirmDelete, mgScreenConfirmRemoveRepo:
		return m.prompt.View()
	case mgScreenAddRepos:
		return m.groupAdd.View()
	case mgScreenDetail:
		if !m.ready {
			v := tea.NewView("")
			v.AltScreen = true
			return v
		}
		v := tea.NewView("\n" + m.detailList.View())
		v.AltScreen = true
		return v
	case mgScreenAddRepoToGroup:
		if !m.ready {
			v := tea.NewView("")
			v.AltScreen = true
			return v
		}
		v := tea.NewView("\n" + m.repoList.View())
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
	// Onboarding: no groups defined yet.
	if len(m.list.Items()) == 0 {
		content := "\n" + titleStyle.Render("Manage groups") + "\n\n" +
			"  No groups defined yet.\n\n" +
			"  Groups let you clone subsets of your repos at once.\n" +
			"  Press a to create your first group.\n\n" +
			"  Press esc to go back.\n"
		v := tea.NewView(content)
		v.AltScreen = true
		return v
	}
	v := tea.NewView("\n" + m.list.View())
	v.AltScreen = true
	return v
}

// newManageGroupModel constructs a manageGroupModel ready for embedding inside
// a parent program (e.g. appModel). The list is built immediately; sizing
// happens when the parent forwards a WindowSizeMsg or sets dimensions directly.
func newManageGroupModel(cfg *config.Config) manageGroupModel {
	m := manageGroupModel{cfg: cfg}
	m.list = m.buildList()
	return m
}

func InitialManageGroupsModel(cfg *config.Config) error {
	m := newManageGroupModel(cfg)
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}
