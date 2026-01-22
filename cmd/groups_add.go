package cmd

import (
	"fmt"
	"reap/config"
	"reap/tui"

	"github.com/spf13/cobra"
)

var addGroupCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new group",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		groupName := args[0]

		cfg, _, err := config.Load()
		if err != nil {
			fmt.Println("Error loading config:", err)
			return
		}

		selectedRepos, err := tui.InitialGroupAddModel(cfg)
		if err != nil {
			return
		}

		for i, repo := range cfg.Repos {
			for _, selected := range selectedRepos {
				if repo.URL == selected {
					cfg.Repos[i].Groups = append(cfg.Repos[i].Groups, config.Group{
						Name:     groupName,
						Selected: true,
					})
				}
			}
		}

		if err := config.Save(cfg); err != nil {
			fmt.Println("Error saving config:", err)
			return
		}

		fmt.Printf("Added group %s to %d repositories.\n", groupName, len(selectedRepos))
	},
}

func init() {
	groupsCmd.AddCommand(addGroupCmd)
}
