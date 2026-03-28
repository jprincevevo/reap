package tui

import (
	"fmt"
	"log"

	"github.com/jprincevevo/reap/config"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

type appScreen int

const (
	appScreenHome     appScreen = iota // group selection — always the anchor
	appScreenRepo                      // repo selection (after group chosen)
	appScreenGroups                    // manageGroupModel
	appScreenRepos                     // manageRepoModel
	appScreenSettings                  // settingsModel
	appScreenPasteAdd                  // paste/drop URL confirmation + group selection
)

// appModel is the single root model passed to tea.NewProgram for the main
// reap command. It hosts every screen for the full lifetime of the process,
// preventing the alt-screen flash that would occur if multiple tea.Programs
// were run back-to-back.
type appModel struct {
	cfg        *config.Config
	screen     appScreen
	home       groupModel
	repo       repoModel
	groups     manageGroupModel
	repos      manageRepoModel
	settings   settingsModel
	pasteAdd   pasteAddModel
	prevScreen appScreen
	width      int
}

func (m appModel) Init() tea.Cmd {
	return nil
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("screen=%d msg=%T %+v", m.screen, msg, msg)

	// Always capture the terminal width so we can immediately size new screens.
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
	}

	// Intercept paste/drop events on every screen: if a valid GitHub URL is
	// detected, transition to the paste-add confirmation flow.
	if paste, ok := msg.(tea.PasteMsg); ok {
		if sanitized, valid := sanitizeGitHubURL(paste.Content); valid {
			preselected := ""
			if m.screen == appScreenGroups && m.groups.screen == mgScreenDetail {
				preselected = m.groups.selectedGroup
			}
			m.pasteAdd = newPasteAddModel(m.cfg, sanitized, preselected)
			if m.width > 0 {
				m.pasteAdd.width = m.width
				m.pasteAdd.groupList.SetSize(m.width, listHeight)
			}
			m.prevScreen = m.screen
			m.screen = appScreenPasteAdd
			return m, nil
		}
	}

	switch m.screen {
	case appScreenHome:
		// Intercept management hotkeys before forwarding to groupModel, but
		// only when not in filter mode (so typing g/r/s in the filter works).
		if kp, ok := msg.(tea.KeyPressMsg); ok && m.home.list.FilterState() != list.Filtering {
			switch kp.String() {
			case "g":
				m.screen = appScreenGroups
				m.groups = newManageGroupModel(m.cfg)
				if m.width > 0 {
					m.groups.width = m.width
					m.groups.list.SetSize(m.width, listHeight)
					m.groups.ready = true
				}
				return m, nil

			case "r":
				m.screen = appScreenRepos
				m.repos = newManageRepoModel(m.cfg)
				if m.width > 0 {
					m.repos.width = m.width
					m.repos.list.SetSize(m.width, listHeight)
					m.repos.ready = true
				}
				return m, nil

			case "s":
				m.screen = appScreenSettings
				m.settings = newSettingsModel(m.cfg)
				return m, nil
			}
		}

		newHome, cmd := m.home.Update(msg)
		m.home = newHome.(groupModel)

		if m.home.choice != "" {
			// User selected a group — transition to repo screen without
			// propagating tea.Quit so the program keeps running.
			m.screen = appScreenRepo
			m.repo = NewRepoModel(m.cfg, m.home.choice)
			if m.width > 0 {
				m.repo.list.SetSize(m.width, listHeight)
				m.repo.ready = true
			}
			return m, nil
		}

		// User quit (or still browsing) — propagate as-is.
		return m, cmd

	case appScreenRepo:
		newRepo, cmd := m.repo.Update(msg)
		m.repo = newRepo.(repoModel)

		if m.repo.goBack {
			// Esc on repo screen — return to home without quitting.
			m.screen = appScreenHome
			m.home = NewGroupModel(m.cfg)
			if m.width > 0 {
				m.home.list.SetSize(m.width, listHeight)
				m.home.ready = true
			}
			return m, nil
		}

		// User confirmed selection, quit explicitly, or still browsing.
		return m, cmd

	case appScreenGroups:
		newGroups, cmd := m.groups.Update(msg)
		m.groups = newGroups.(manageGroupModel)

		if m.groups.goBack {
			// User finished with group management — rebuild home to reflect
			// any config changes made during the session.
			m.screen = appScreenHome
			m.home = NewGroupModel(m.cfg)
			if m.width > 0 {
				m.home.list.SetSize(m.width, listHeight)
				m.home.ready = true
			}
			return m, nil
		}

		return m, cmd

	case appScreenRepos:
		newRepos, cmd := m.repos.Update(msg)
		m.repos = newRepos.(manageRepoModel)

		if m.repos.goBack {
			m.screen = appScreenHome
			m.home = NewGroupModel(m.cfg)
			if m.width > 0 {
				m.home.list.SetSize(m.width, listHeight)
				m.home.ready = true
			}
			return m, nil
		}

		return m, cmd

	case appScreenSettings:
		newSettings, cmd := m.settings.Update(msg)
		m.settings = newSettings.(settingsModel)

		if m.settings.quitting {
			return m, cmd
		}
		if m.settings.goBack {
			m.screen = appScreenHome
			m.home = NewGroupModel(m.cfg)
			if m.width > 0 {
				m.home.list.SetSize(m.width, listHeight)
				m.home.ready = true
			}
			return m, nil
		}

		return m, cmd

	case appScreenPasteAdd:
		newPA, cmd := m.pasteAdd.Update(msg)
		m.pasteAdd = newPA.(pasteAddModel)

		if m.pasteAdd.done || m.pasteAdd.cancelled {
			// Rebuild home to reflect any config changes, then return there.
			m.screen = appScreenHome
			m.home = NewGroupModel(m.cfg)
			if m.width > 0 {
				m.home.list.SetSize(m.width, listHeight)
				m.home.ready = true
			}
			return m, nil // discard the tea.Quit the child sent
		}

		return m, cmd
	}

	return m, nil
}

func (m appModel) View() tea.View {
	switch m.screen {
	case appScreenHome:
		return m.home.View()
	case appScreenRepo:
		return m.repo.View()
	case appScreenGroups:
		return m.groups.View()
	case appScreenRepos:
		return m.repos.View()
	case appScreenSettings:
		return m.settings.View()
	case appScreenPasteAdd:
		return m.pasteAdd.View()
	}
	return tea.NewView("")
}

// InitialAppModel runs a single persistent tea.Program that hosts all screens
// for the lifetime of the reap command. It returns the selected repository
// URLs when the user confirms a clone, or an error if the user aborted.
func InitialAppModel(cfg *config.Config) ([]string, error) {
	m := appModel{
		cfg:    cfg,
		screen: appScreenHome,
		home:   NewGroupModel(cfg),
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	fm, ok := finalModel.(appModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	// Only a confirmed repo selection (enter on the repo screen) results in
	// cloning. Any other exit path (user quit, esc back to home, etc.) returns
	// an "aborted" error so the caller skips cloning.
	if fm.screen != appScreenRepo || fm.repo.quitting {
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
