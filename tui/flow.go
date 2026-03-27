package tui

import (
	"fmt"

	"github.com/jprincevevo/reap/config"

	tea "charm.land/bubbletea/v2"
)

type flowScreen int

const (
	flowScreenGroup flowScreen = iota
	flowScreenRepo
)

// flowModel runs group selection and repo selection inside a single
// tea.Program so that the terminal never exits alt screen between the two
// screens, eliminating the visible flash.
type flowModel struct {
	cfg    *config.Config
	screen flowScreen
	group  groupModel
	repo   repoModel
	width  int // last known terminal width, used when sizing new screens
}

func (m flowModel) Init() tea.Cmd {
	return nil
}

func (m flowModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Always capture the terminal width so we can immediately size new screens.
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
	}

	switch m.screen {
	case flowScreenGroup:
		newGroup, cmd := m.group.Update(msg)
		m.group = newGroup.(groupModel)

		if m.group.choice != "" {
			// User selected a group — transition to repo screen without
			// propagating tea.Quit so the program keeps running.
			m.screen = flowScreenRepo
			m.repo = NewRepoModel(m.cfg, m.group.choice)
			if m.width > 0 {
				m.repo.list.SetSize(m.width, listHeight)
				m.repo.ready = true
			}
			return m, nil
		}
		// User quit (or still browsing) — propagate the command as-is.
		return m, cmd

	case flowScreenRepo:
		newRepo, cmd := m.repo.Update(msg)
		m.repo = newRepo.(repoModel)

		if m.repo.goBack {
			// User pressed esc — transition back to group screen without
			// propagating tea.Quit so the program keeps running.
			m.screen = flowScreenGroup
			m.group = NewGroupModel(m.cfg)
			if m.width > 0 {
				m.group.list.SetSize(m.width, listHeight)
				m.group.ready = true
			}
			return m, nil
		}
		// User confirmed, quit, or still browsing — propagate.
		return m, cmd
	}

	return m, nil
}

func (m flowModel) View() tea.View {
	switch m.screen {
	case flowScreenGroup:
		return m.group.View()
	case flowScreenRepo:
		return m.repo.View()
	}
	return tea.NewView("")
}

// InitialFlowModel runs a single program that covers group selection followed
// by repo selection. Because both screens share one program, the alt screen
// never exits between them and there is no visible flash on transition.
func InitialFlowModel(cfg *config.Config) ([]string, error) {
	m := flowModel{
		cfg:    cfg,
		screen: flowScreenGroup,
		group:  NewGroupModel(cfg),
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	fm, ok := finalModel.(flowModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	// If the program exited while still on the group screen, or the user
	// aborted on the repo screen, there is nothing to clone.
	if fm.screen == flowScreenGroup || fm.repo.quitting {
		return nil, fmt.Errorf("aborted")
	}

	var selected []string
	for _, listItem := range fm.repo.list.Items() {
		if ri, ok := listItem.(repoItem); ok && ri.selected {
			selected = append(selected, ri.url)
		}
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("no repositories selected")
	}

	return selected, nil
}
