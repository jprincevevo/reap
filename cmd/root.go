package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/creativeprojects/go-selfupdate"
	"github.com/jprincevevo/reap/config"
	"github.com/jprincevevo/reap/tui"
	"github.com/jprincevevo/reap/version"

	"github.com/spf13/cobra"
)

var depth int
var showVersion bool
var cloneDir string
var pullFlag bool

var rootCmd = &cobra.Command{
	Use:   "reap",
	Short: "A CLI tool for batch-cloning repositories",
	Long:  `reap is a terminal user interface application for cloning git repositories from a yaml config file.`,
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			fmt.Printf("reap version %s\n", version.Version)
			return
		}

		updateDone := make(chan struct{})
		go func() {
			checkForUpdates()
			close(updateDone)
		}()

		cfg, created, err := config.Load()
		if err != nil {
			fmt.Println("Error loading config:", err)
			os.Exit(1)
		}

		if created {
			configPath, _ := config.GetConfigPath()
			fmt.Printf("Created default config file at %s\n", configPath)
		}

		if len(cfg.Repos) == 0 {
			fmt.Println("No repositories configured. Add one with `reap repo add <url>`.")
			return
		}

		if depth == 0 && cfg.DefaultDepth > 0 {
			depth = cfg.DefaultDepth
		}

		if len(args) > 0 {
			<-updateDone
			cloneRepos(args, depth, cloneDir, pullFlag)
			return
		}

		<-updateDone

		var selected []string
		if cfg.HasGroups() {
			selected, err = tui.InitialFlowModel(cfg)
		} else {
			selected, err = tui.InitialRepoModel(cfg, "Show All")
		}
		if err != nil {
			return
		}

		if len(selected) > 0 {
			cloneRepos(selected, depth, cloneDir, pullFlag)
		}
	},
}

func init() {
	rootCmd.Flags().IntVar(&depth, "depth", 0, "Set the clone depth")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Display version")
	rootCmd.Flags().StringVar(&cloneDir, "dir", "", "Clone repositories into this directory")
	rootCmd.Flags().BoolVar(&pullFlag, "pull", false, "Pull instead of clone for existing repos")
}

func cloneRepos(repos []string, depth int, dir string, pull bool) {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err == nil {
		confirmed, err := tui.InitialConfirmModel("This directory is a git repository. Continue?")
		if err != nil || !confirmed {
			fmt.Println("Aborting.")
			return
		}
	}

	if err := tui.InitialCloneModel(repos, depth, dir, pull); err != nil {
		fmt.Println("Error cloning repositories:", err)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func checkForUpdates() {
	updater, err := selfupdate.NewUpdater(selfupdate.Config{})
	if err != nil {
		return
	}

	repo := selfupdate.ParseSlug("jprincevevo/reap")

	latest, found, err := updater.DetectLatest(context.Background(), repo)
	if err != nil {
		return
	}

	if found && latest.Version() != version.Version && version.Version != "dev" {
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingLeft(2).
			PaddingRight(2)

		msg := fmt.Sprintf("✨ A new version of reap is available: %s", latest.Version())
		fmt.Printf("\n%s\n\n", style.Render(msg))
		time.Sleep(2 * time.Second)
	}
}
