package cmd

import (
	"fmt"
	"reap/config"

	"github.com/spf13/cobra"
)

var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "Manage repository groups",
}

var listGroupsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all custom groups",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _, err := config.Load()
		if err != nil {
			fmt.Println("Error loading config:", err)
			return
		}

		seenGroups := make(map[string]bool)
		for _, repo := range cfg.Repos {
			for _, group := range repo.Groups {
				if !seenGroups[group.Name] {
					seenGroups[group.Name] = true
					fmt.Println(group.Name)
				}
			}
		}
	},
}

func init() {
	groupsCmd.AddCommand(listGroupsCmd)
	rootCmd.AddCommand(groupsCmd)
}
