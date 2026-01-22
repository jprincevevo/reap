package cmd

import (
	"fmt"
	"reap/config"
	"reap/tui"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a repository from the config",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _, err := config.Load()
		if err != nil {
			fmt.Println("Error loading config:", err)
			return
		}

		repoToRemove, err := tui.InitialRemoveModel(cfg)
		if err != nil {
			return
		}

		var newRepos []config.Repo
		for _, repo := range cfg.Repos {
			if repo.URL != repoToRemove {
				newRepos = append(newRepos, repo)
			}
		}
		cfg.Repos = newRepos

		if err := config.Save(cfg); err != nil {
			fmt.Println("Error saving config:", err)
			return
		}

		fmt.Printf("Removed %s from the config.\n", repoToRemove)
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
