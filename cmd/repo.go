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
		cfg, _, err := config.Load()
		if err != nil {
			fmt.Println("Error loading config:", err)
			return
		}

		if err := tui.InitialManageReposModel(cfg); err != nil {
			fmt.Println("Error:", err)
		}
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

		if !cfg.AddRepo(args[0]) {
			fmt.Printf("%s is already in the config.\n", args[0])
			return
		}

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

		if len(cfg.Repos) == 0 {
			fmt.Println("No repositories configured.")
			return
		}

		repoToRemove, err := tui.InitialRemoveModel(cfg)
		if err != nil {
			return
		}

		cfg.RemoveRepo(repoToRemove)

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

		if len(cfg.Repos) == 0 {
			fmt.Println("No repositories configured.")
			return
		}

		for _, r := range cfg.Repos {
			fmt.Println(r.URL)
		}
	},
}

func init() {
	repoCmd.AddCommand(addRepoCmd)
	repoCmd.AddCommand(removeRepoCmd)
	repoCmd.AddCommand(listReposCmd)
	rootCmd.AddCommand(repoCmd)
}
