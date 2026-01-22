package cmd

import (
	"fmt"

	"github.com/jprincevevo/reap/config"
	"github.com/jprincevevo/reap/tui"

	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var addRepoCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a new repository to the config",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _, err := config.Load()
		if err != nil {
			fmt.Println("Error loading config:", err)
			return
		}

		newRepo := config.Repo{
			URL:      args[0],
			Selected: true,
		}

		cfg.Repos = append(cfg.Repos, newRepo)

		if err := config.Save(cfg); err != nil {
			fmt.Println("Error saving config:", err)
			return
		}

		fmt.Printf("Added %s to the config.\n", args[0])
	},
}

var removeRepoCmd = &cobra.Command{
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

var listReposCmd = &cobra.Command{
	Use:   "list",
	Short: "List all repositories in the config",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _, err := config.Load()
		if err != nil {
			fmt.Println("Error loading config:", err)
			return
		}

		for _, repo := range cfg.Repos {
			fmt.Println(repo.URL)
		}
	},
}

func init() {
	repoCmd.AddCommand(addRepoCmd)
	repoCmd.AddCommand(removeRepoCmd)
	repoCmd.AddCommand(listReposCmd)
	rootCmd.AddCommand(repoCmd)
}
