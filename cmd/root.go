package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/creativeprojects/go-selfupdate"
	"github.com/jprincevevo/reap/config"
	"github.com/jprincevevo/reap/tui"
	"github.com/jprincevevo/reap/version"

	"github.com/spf13/cobra"
)

var depth int
var showVersion bool

var rootCmd = &cobra.Command{
	Use:   "reap",
	Short: "A CLI tool for batch-cloning repositories",
	Long:  `reap is a terminal user interface application for cloning git repositories from a yaml config file.`,
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			fmt.Printf("reap version %s\n", version.Version)
			return
		}

		go checkForUpdates()
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

		if len(args) > 0 {
			cloneRepos(args, depth)
			return
		}

		var group string
		if cfg.HasGroups() {
			g, err := tui.InitialGroupModel(cfg)
			if err != nil {
				return
			}
			group = g
		} else {
			group = "Show All"
		}

		selected, err := tui.InitialRepoModel(cfg, group)
		if err != nil {
			return
		}

		if len(selected) > 0 {
			cloneRepos(selected, depth)
		}
	},
}

func init() {
	rootCmd.Flags().IntVar(&depth, "depth", 0, "Set the clone depth")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Display version")
}

func cloneRepos(repos []string, depth int) {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err == nil {
		confirmed, err := tui.InitialConfirmModel("This directory is a git repository. Continue?")
		if err != nil || !confirmed {
			fmt.Println("Aborting.")
			return
		}
	}

	if err := tui.InitialCloneModel(repos, depth); err != nil {
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

	// Compare versions
	if found && latest.Version() != version.Version {
		// Define a cleaner style without vertical padding
		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingLeft(2).
			PaddingRight(2)

		// Create the message
		msg := fmt.Sprintf("✨ A new version of reap is available: %s", latest.Version())

		// Render it. We add the vertical spacing manually in Printf
		// to ensure the cursor stays at the left margin.
		fmt.Printf("\n%s\n\n", style.Render(msg))

		// Give the user time to see it before the TUI takes over the screen
		time.Sleep(2 * time.Second)
	}
}