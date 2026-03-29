package tui

import (
	"fmt"
	"strconv"

	"charm.land/bubbles/v2/textinput"
	"github.com/jprincevevo/reap/config"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type settingsField int

const (
	settingsFieldDepth settingsField = iota
	settingsFieldDir
	settingsFieldPull
	settingsFieldCount // sentinel for wrap-around
)

// settingsModel presents all three config fields on a single screen. Focus
// cycles with Tab/Shift-Tab. Enter saves every field to config and returns
// home. Esc discards all pending changes.
type settingsModel struct {
	cfg        *config.Config
	focused    settingsField
	depthInput textinput.Model
	dirInput   textinput.Model
	pull       bool
	goBack     bool
	quitting   bool
}

func newSettingsModel(cfg *config.Config) settingsModel {
	depth := textinput.New()
	depth.Placeholder = "0"
	depth.Prompt = "  > "
	depth.SetWidth(40)
	if cfg.DefaultDepth > 0 {
		depth.SetValue(fmt.Sprintf("%d", cfg.DefaultDepth))
	}
	depth.Focus()

	dir := textinput.New()
	dir.Placeholder = "current directory"
	dir.Prompt = "  > "
	dir.SetWidth(40)
	if cfg.DefaultDir != "" {
		dir.SetValue(cfg.DefaultDir)
	}

	return settingsModel{
		cfg:        cfg,
		focused:    settingsFieldDepth,
		depthInput: depth,
		dirInput:   dir,
		pull:       cfg.DefaultPull,
	}
}

func (m settingsModel) Init() tea.Cmd {
	return nil
}

func (m *settingsModel) setFocus(f settingsField) {
	m.focused = f
	if f == settingsFieldDepth {
		m.depthInput.Focus()
		m.dirInput.Blur()
	} else if f == settingsFieldDir {
		m.depthInput.Blur()
		m.dirInput.Focus()
	} else {
		m.depthInput.Blur()
		m.dirInput.Blur()
	}
}

func (m settingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch kp.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "esc":
			m.goBack = true
			return m, tea.Quit

		case "tab", "down":
			next := settingsField((int(m.focused) + 1) % int(settingsFieldCount))
			m.setFocus(next)
			return m, nil

		case "shift+tab", "up":
			prev := settingsField((int(m.focused) - 1 + int(settingsFieldCount)) % int(settingsFieldCount))
			m.setFocus(prev)
			return m, nil

		case "space":
			if m.focused == settingsFieldPull {
				m.pull = !m.pull
				return m, nil
			}

		case "enter":
			d, err := strconv.Atoi(m.depthInput.Value())
			if err != nil {
				d = 0
			}
			m.cfg.DefaultDepth = d
			m.cfg.DefaultDir = m.dirInput.Value()
			m.cfg.DefaultPull = m.pull
			logSaveErr(config.Save(m.cfg))
			m.goBack = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	switch m.focused {
	case settingsFieldDepth:
		m.depthInput, cmd = m.depthInput.Update(msg)
	case settingsFieldDir:
		m.dirInput, cmd = m.dirInput.Update(msg)
	}
	return m, cmd
}

func (m settingsModel) View() tea.View {
	if m.quitting || m.goBack {
		return tea.NewView("")
	}

	labelStyle := lipgloss.NewStyle().Foreground(colorDim).MarginLeft(2)
	focusedLabelStyle := lipgloss.NewStyle().Bold(true).Foreground(colorAccent).MarginLeft(2)

	depthLabel := labelStyle.Render("Default clone depth (0 = full history)")
	dirLabel := labelStyle.Render("Default clone directory (empty = current directory)")
	pullLabel := labelStyle.Render("Pull instead of clone")

	if m.focused == settingsFieldDepth {
		depthLabel = focusedLabelStyle.Render("Default clone depth (0 = full history)")
	} else if m.focused == settingsFieldDir {
		dirLabel = focusedLabelStyle.Render("Default clone directory (empty = current directory)")
	} else {
		pullLabel = focusedLabelStyle.Render("Pull instead of clone")
	}

	pullBadge := selectBadge(m.pull)
	pullStatus := " Pull (off)"
	if m.pull {
		pullStatus = " Pull (on)"
	}

	dimStyle := lipgloss.NewStyle().Foreground(colorDim)
	keyStyle := lipgloss.NewStyle().Foreground(colorMuted)
	sep := dimStyle.Render(" • ")

	helpLine := lipgloss.NewStyle().MarginLeft(2).Render(
		keyStyle.Render("tab") + dimStyle.Render(" next") + sep +
			keyStyle.Render("shift+tab") + dimStyle.Render(" prev") + sep +
			keyStyle.Render("enter") + dimStyle.Render(" save") + sep +
			keyStyle.Render("esc") + dimStyle.Render(" back"),
	)

	content := "\n" + titleStyle.Render("Settings") + "\n\n" +
		depthLabel + "\n" +
		"  " + m.depthInput.View() + "\n\n" +
		dirLabel + "\n" +
		"  " + m.dirInput.View() + "\n\n" +
		pullLabel + "\n" +
		"  " + pullBadge + pullStatus + "\n\n" +
		helpLine + "\n"

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
