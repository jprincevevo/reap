package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type status int

const (
	cloning status = iota
	done
	failed
)

type repoState struct {
	url    string
	status status
	err    error
}

type cloneMsg struct{ repoIndex int }
type errMsg struct {
	repoIndex int
	err       error
}
type finishedMsg struct{}

type cloneModel struct {
	repos    []repoState
	spinner  spinner.Model
	quitting bool
	wg       *sync.WaitGroup
	depth    int
}

func (m cloneModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.runClones)
}

func (m cloneModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	case cloneMsg:
		m.repos[msg.repoIndex].status = done
		return m, nil
	case errMsg:
		m.repos[msg.repoIndex].status = failed
		m.repos[msg.repoIndex].err = msg.err
		return m, nil
	case finishedMsg:
		m.quitting = true
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m cloneModel) View() string {
	if m.quitting {
		var s strings.Builder
		s.WriteString("Cloning finished.\n\n")
		for _, repo := range m.repos {
			switch repo.status {
			case done:
				s.WriteString(fmt.Sprintf("✓ %s\n", repo.url))
			case failed:
				s.WriteString(fmt.Sprintf("✗ %s: %s\n", repo.url, repo.err))
			}
		}
		return s.String()
	}

	var s strings.Builder
	s.WriteString("Cloning repositories...\n\n")

	for _, repo := range m.repos {
		switch repo.status {
		case cloning:
			s.WriteString(fmt.Sprintf("%s %s\n", m.spinner.View(), repo.url))
		case done:
			s.WriteString(fmt.Sprintf("✓ %s\n", repo.url))
		case failed:
			s.WriteString(fmt.Sprintf("✗ %s: %s\n", repo.url, repo.err))
		}
	}

	return s.String()
}

func (m *cloneModel) cloneRepo(repoIndex int) {
	defer m.wg.Done()
	repo := m.repos[repoIndex]
	repoName := strings.TrimSuffix(filepath.Base(repo.url), ".git")
	if _, err := os.Stat(repoName); !os.IsNotExist(err) {
		p.Send(errMsg{repoIndex, fmt.Errorf("directory %s already exists", repoName)})
		return
	}

	args := []string{"clone", repo.url}
	if m.depth > 0 {
		args = append(args, "--depth", fmt.Sprintf("%d", m.depth))
	}

	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		p.Send(errMsg{repoIndex, err})
		return
	}

	p.Send(cloneMsg{repoIndex})
}

func (m *cloneModel) runClones() tea.Msg {
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

func InitialCloneModel(repos []string, depth int) error {
	repoStates := make([]repoState, len(repos))
	for i, repo := range repos {
		repoStates[i] = repoState{url: repo, status: cloning}
	}

	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := cloneModel{
		repos:   repoStates,
		spinner: s,
		wg:      &sync.WaitGroup{},
		depth:   depth,
	}

	p = tea.NewProgram(m)

	_, err := p.Run()
	return err
}
