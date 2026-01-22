package cmd

import (
	"fmt"
	"os"
	"reap/config"
	"reap/tui"

	"github.com/spf13/cobra"
)

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
			fmt.Println("Created default config file at ~/.reap.yaml")
			cmd.Help()
			return
		}

		if len(args) > 0 {
			cloneRepos(args)
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
			cloneRepos(selected)
		}
	},
}

func cloneRepos(repos []string) {
	if _, err := os.Stat(".git"); !os.IsNotExist(err) {
		fmt.Println("This directory is already a git repository. Are you sure you want to continue? (y/n)")
		var response string
		fmt.Scanln(&response)
		if response != "y" {
			fmt.Println("Aborting.")
			return
		}
	}

	if err := tui.InitialCloneModel(repos); err != nil {
		fmt.Println("Error cloning repositories:", err)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
