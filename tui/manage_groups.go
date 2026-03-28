package tui

import (
	"fmt"

	"github.com/jprincevevo/reap/config"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

func removeGroupFromAllRepos(cfg *config.Config, groupName string) int {
	removed := 0
	for i, repo := range cfg.Repos {
		var kept []config.Group
		for _, g := range repo.Groups {
			if g.Name != groupName {
				kept = append(kept, g)
			} else {
				removed++
			}
		}
		cfg.Repos[i].Groups = kept
	}
	return removed
}

func applyGroupToRepos(cfg *config.Config, groupName string, selectedURLs []string) int {
	selected := make(map[string]bool, len(selectedURLs))
	for _, u := range selectedURLs {
		selected[u] = true
	}

	modified := 0
	for i, repo := range cfg.Repos {
		if !selected[repo.URL] {
			continue
		}
		alreadyHas := false
		for _, g := range repo.Groups {
			if g.Name == groupName {
				alreadyHas = true
				break
			}
		}
		if !alreadyHas {
			cfg.Repos[i].Groups = append(cfg.Repos[i].Groups, config.Group{
				Name:     groupName,
				Selected: true,
			})
			modified++
		}
	}
	return modified
}

type manageGroupScreen int

const (
	mgScreenList manageGroupScreen = iota
	mgScreenAdd
	mgScreenAddRepos
	mgScreenRename
	mgScreenConfirmDelete
)

type manageGroupModel struct {
	cfg           *config.Config
	screen        manageGroupScreen
	list          list.Model
	prompt        promptModel
	groupAdd      groupAddModel
	width         int
	ready         bool
	done          bool
	pendingGroup  string
	selectedGroup string
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

	const defaultWidth = 20
	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Manage groups"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename")),
			key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
			key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "done")),
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
		case "ctrl+c", "q", "enter":
			m.done = true
			return m, tea.Quit

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
			applyGroupToRepos(m.cfg, m.pendingGroup, selectedURLs)
			if err := config.Save(m.cfg); err != nil {
				fmt.Println("Error saving config:", err)
			}
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
			if err := config.Save(m.cfg); err != nil {
				fmt.Println("Error saving config:", err)
			}
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
			removeGroupFromAllRepos(m.cfg, m.selectedGroup)
			if err := config.Save(m.cfg); err != nil {
				fmt.Println("Error saving config:", err)
			}
		}
		m.screen = mgScreenList
		m.list = m.buildList()
		return m, nil
	}

	return m, cmd
}

func (m manageGroupModel) View() tea.View {
	switch m.screen {
	case mgScreenAdd, mgScreenRename, mgScreenConfirmDelete:
		return m.prompt.View()
	case mgScreenAddRepos:
		return m.groupAdd.View()
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

func InitialManageGroupsModel(cfg *config.Config) error {
	m := manageGroupModel{
		cfg: cfg,
	}
	m.list = m.buildList()

	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}
