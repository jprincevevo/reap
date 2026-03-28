package tui

import (
	"fmt"
	"strconv"

	"github.com/jprincevevo/reap/config"

	tea "charm.land/bubbletea/v2"
)

type settingsField int

const (
	settingsFieldDepth settingsField = iota // integer: default clone depth
	settingsFieldDir                        // string: default clone directory
	settingsFieldPull                       // bool: pull instead of clone
)

// settingsModel presents three config fields in sequence, each confirmed with
// enter. Changes are written immediately via config.Save on confirm. Esc on
// any field returns to appScreenHome without saving that field's pending change.
type settingsModel struct {
	cfg      *config.Config
	field    settingsField
	prompt   promptModel // used for depth and dir fields
	pull     bool        // current toggle value for the pull field
	goBack   bool
	quitting bool
}

func newSettingsModel(cfg *config.Config) settingsModel {
	prompt := NewPromptModel("Default clone depth (0 = full history)", "0")
	if cfg.DefaultDepth > 0 {
		prompt.input.SetValue(fmt.Sprintf("%d", cfg.DefaultDepth))
	}
	return settingsModel{
		cfg:    cfg,
		field:  settingsFieldDepth,
		pull:   cfg.DefaultPull,
		prompt: prompt,
	}
}

func (m settingsModel) Init() tea.Cmd {
	return nil
}

func (m settingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Intercept universal exit keys before forwarding to any sub-model.
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch kp.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			m.goBack = true
			return m, tea.Quit
		case "q":
			// Only quit on "q" for the pull toggle, not while typing in a text field.
			if m.field == settingsFieldPull {
				m.goBack = true
				return m, tea.Quit
			}
		}
	}

	switch m.field {
	case settingsFieldDepth, settingsFieldDir:
		newPrompt, cmd := m.prompt.Update(msg)
		m.prompt = newPrompt.(promptModel)

		if m.prompt.quitting {
			if m.field == settingsFieldDepth {
				d, err := strconv.Atoi(m.prompt.input.Value())
				if err != nil {
					d = 0
				}
				m.cfg.DefaultDepth = d
				logSaveErr(config.Save(m.cfg))
				m.field = settingsFieldDir
				m.prompt = NewPromptModel("Default clone directory (empty = current directory)", "")
				if m.cfg.DefaultDir != "" {
					m.prompt.input.SetValue(m.cfg.DefaultDir)
				}
			} else {
				m.cfg.DefaultDir = m.prompt.input.Value()
				logSaveErr(config.Save(m.cfg))
				m.field = settingsFieldPull
			}
			return m, nil
		}

		return m, cmd

	case settingsFieldPull:
		if kp, ok := msg.(tea.KeyPressMsg); ok {
			switch kp.String() {
			case "space":
				m.pull = !m.pull
				return m, nil
			case "enter":
				m.cfg.DefaultPull = m.pull
				logSaveErr(config.Save(m.cfg))
				m.goBack = true
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m settingsModel) View() tea.View {
	if m.quitting || m.goBack {
		return tea.NewView("")
	}

	switch m.field {
	case settingsFieldDepth, settingsFieldDir:
		return m.prompt.View()

	case settingsFieldPull:
		pullLabel := " Pull (off)"
		if m.pull {
			pullLabel = " Pull (on)"
		}
		badge := selectBadge(m.pull) + pullLabel
		content := "\n" + titleStyle.Render("Pull existing repos") + "\n\n" +
			"  " + badge + "\n\n" +
			"  space toggle  enter confirm  esc back\n"
		v := tea.NewView(content)
		v.AltScreen = true
		return v
	}

	return tea.NewView("")
}
