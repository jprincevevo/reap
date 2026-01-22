package cmd

import (
	"fmt"
	"reap/config"

	"github.com/spf13/cobra"
)

var removeGroupCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a group from all repositories",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		groupName := args[0]

		cfg, _, err := config.Load()
		if err != nil {
			fmt.Println("Error loading config:", err)
			return
		}

		for i, repo := range cfg.Repos {
			var newGroups []config.Group
			for _, group := range repo.Groups {
				if group.Name != groupName {
					newGroups = append(newGroups, group)
				}
			}
			cfg.Repos[i].Groups = newGroups
		}

		if err := config.Save(cfg); err != nil {
			fmt.Println("Error saving config:", err)
			return
		}

		fmt.Printf("Removed group %s from all repositories.\n", groupName)
	},
}

func init() {
	groupsCmd.AddCommand(removeGroupCmd)
}
