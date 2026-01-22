package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"reap/config"
	"reap/tui"

	"github.com/spf13/cobra"
)

var depth int

var rootCmd = &cobra.Command{
	Use:   "reap",
	Short: "A CLI tool for batch-cloning repositories",
	Long:  `reap is a terminal user interface application for cloning git repositories from a yaml config file.`,
	Run: func(cmd *cobra.Command, args []string) {
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
