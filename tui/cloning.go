package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/stopwatch"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	cloneSuccessStyle = lipgloss.NewStyle().Foreground(colorSuccess)
	cloneErrorStyle   = lipgloss.NewStyle().Foreground(colorError)
	cloneHeaderStyle  = lipgloss.NewStyle().Bold(true)
	cloneSummaryStyle = lipgloss.NewStyle().Foreground(colorMuted)
)

type status int

const (
	cloning status = iota
	done
	failed
)

type repoState struct {
	url     string
	status  status
	pulling bool
	err     error
}

type cloneMsg struct{ repoIndex int }
type errMsg struct {
	repoIndex int
	err       error
}
type finishedMsg struct{}

type cloneModel struct {
	repos     []repoState
	spinner   spinner.Model
	progress  progress.Model
	stopwatch stopwatch.Model
	quitting  bool
	wg        *sync.WaitGroup
	depth     int
	dir       string
	pull      bool
}

func (m cloneModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.stopwatch.Init(), m.runClones)
}

func (m cloneModel) completedPercent() float64 {
	n := 0
	for _, r := range m.repos {
		if r.status == done || r.status == failed {
			n++
		}
	}
	return float64(n) / float64(len(m.repos))
}

func (m cloneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	case cloneMsg:
		m.repos[msg.repoIndex].status = done
		cmd := m.progress.SetPercent(m.completedPercent())
		return m, cmd
	case errMsg:
		m.repos[msg.repoIndex].status = failed
		m.repos[msg.repoIndex].err = msg.err
		cmd := m.progress.SetPercent(m.completedPercent())
		return m, cmd
	case finishedMsg:
		m.quitting = true
		return m, tea.Batch(m.stopwatch.Stop(), tea.Quit)
	case progress.FrameMsg:
		pm, cmd := m.progress.Update(msg)
		m.progress = pm
		return m, cmd
	case stopwatch.TickMsg:
		sw, cmd := m.stopwatch.Update(msg)
		m.stopwatch = sw
		return m, cmd
	case stopwatch.StartStopMsg:
		sw, cmd := m.stopwatch.Update(msg)
		m.stopwatch = sw
		return m, cmd
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func formatElapsed(d time.Duration) string {
	d = d.Round(100 * time.Millisecond)
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return d.Truncate(time.Second).String()
}

func (m cloneModel) View() tea.View {
	var s strings.Builder
	if m.quitting {
		s.WriteString(cloneHeaderStyle.Render("Cloning complete.") + "\n\n")
		var succeeded, failCount int
		for _, repo := range m.repos {
			switch repo.status {
			case done:
				succeeded++
				s.WriteString(cloneSuccessStyle.Render("✓") + " " + repo.url + "\n")
			case failed:
				failCount++
				s.WriteString(cloneErrorStyle.Render("✗") + " " + repo.url + ": " + repo.err.Error() + "\n")
			}
		}
		var parts []string
		if succeeded > 0 {
			parts = append(parts, fmt.Sprintf("%d cloned", succeeded))
		}
		if failCount > 0 {
			parts = append(parts, fmt.Sprintf("%d failed", failCount))
		}
		parts = append(parts, formatElapsed(m.stopwatch.Elapsed()))
		if len(parts) > 0 {
			s.WriteString("\n" + cloneSummaryStyle.Render(strings.Join(parts, " · ")) + "\n")
		}
	} else {
		elapsed := formatElapsed(m.stopwatch.Elapsed())
		s.WriteString(fmt.Sprintf("Cloning repositories...  %s\n\n", cloneSummaryStyle.Render(elapsed)))
		s.WriteString(m.progress.View() + "\n\n")
		for _, repo := range m.repos {
			switch repo.status {
			case cloning:
				if repo.pulling {
					s.WriteString(cloneSummaryStyle.Render("↑") + " " + repo.url + "\n")
				} else {
					s.WriteString(fmt.Sprintf("%s %s\n", m.spinner.View(), repo.url))
				}
			case done:
				s.WriteString(cloneSuccessStyle.Render("✓") + " " + repo.url + "\n")
			case failed:
				s.WriteString(cloneErrorStyle.Render("✗") + " " + repo.url + ": " + repo.err.Error() + "\n")
			}
		}
	}
	return tea.NewView(s.String())
}

func (m *cloneModel) targetDir(url string) string {
	repoName := strings.TrimSuffix(filepath.Base(url), ".git")
	if m.dir != "" {
		return filepath.Join(m.dir, repoName)
	}
	return repoName
}

func (m *cloneModel) cloneRepo(repoIndex int) {
	defer m.wg.Done()
	repo := m.repos[repoIndex]
	target := m.targetDir(repo.url)

	if repo.pulling {
		cmd := exec.Command("git", "-C", target, "pull")
		if err := cmd.Run(); err != nil {
			p.Send(errMsg{repoIndex, err})
			return
		}
		p.Send(cloneMsg{repoIndex})
		return
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		p.Send(errMsg{repoIndex, fmt.Errorf("directory %s already exists", target)})
		return
	}

	args := []string{"clone", repo.url}
	if m.depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", m.depth))
	}
	if m.dir != "" {
		args = append(args, target)
	}

	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		p.Send(errMsg{repoIndex, err})
		return
	}

	p.Send(cloneMsg{repoIndex})
}

func (m *cloneModel) runClones() tea.Msg {
	if m.dir != "" {
		if err := os.MkdirAll(m.dir, 0755); err != nil {
			for i := range m.repos {
				p.Send(errMsg{i, fmt.Errorf("failed to create directory %s: %w", m.dir, err)})
			}
			return nil
		}
	}

	repoChannel := make(chan int, len(m.repos))

	for i := 0; i < 5; i++ {
		go func() {
			for repoIndex := range repoChannel {
				m.cloneRepo(repoIndex)
			}
		}()
	}

	for i := range m.repos {
		m.wg.Add(1)
		repoChannel <- i
	}
	close(repoChannel)

	go func() {
		m.wg.Wait()
		p.Send(finishedMsg{})
	}()

	return nil
}

var p *tea.Program

func InitialCloneModel(repos []string, depth int, dir string, pull bool) error {
	repoStates := make([]repoState, len(repos))
	for i, repo := range repos {
		rs := repoState{url: repo, status: cloning}
		if pull {
			repoName := strings.TrimSuffix(filepath.Base(repo), ".git")
			target := repoName
			if dir != "" {
				target = filepath.Join(dir, repoName)
			}
			if info, err := os.Stat(filepath.Join(target, ".git")); err == nil && info.IsDir() {
				rs.pulling = true
			}
		}
		repoStates[i] = rs
	}

	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(colorMuted)

	prog := progress.New(
		progress.WithColors(colorAccent, colorSuccess),
		progress.WithoutPercentage(),
	)

	sw := stopwatch.New(stopwatch.WithInterval(100 * time.Millisecond))

	m := cloneModel{
		repos:     repoStates,
		spinner:   s,
		progress:  prog,
		stopwatch: sw,
		wg:        &sync.WaitGroup{},
		depth:     depth,
		dir:       dir,
		pull:      pull,
	}

	p = tea.NewProgram(m)

	_, err := p.Run()
	return err
}
