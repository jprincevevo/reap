package cmd

import (
	"fmt"

	"github.com/jprincevevo/reap/config"
	"github.com/jprincevevo/reap/tui"

	"github.com/spf13/cobra"
)

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage repository groups",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _, err := config.Load()
		if err != nil {
			fmt.Println("Error loading config:", err)
			return
		}

		if err := tui.InitialManageGroupsModel(cfg); err != nil {
			fmt.Println("Error:", err)
		}
	},
}

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

		modified := applyGroupToRepos(cfg, groupName, selectedRepos)

		if err := config.Save(cfg); err != nil {
			fmt.Println("Error saving config:", err)
			return
		}

		fmt.Printf("Added group %s to %d repositories.\n", groupName, modified)
	},
}

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

		if removeGroupFromAllRepos(cfg, groupName) == 0 {
			fmt.Printf("Group %q not found in any repository.\n", groupName)
			return
		}

		if err := config.Save(cfg); err != nil {
			fmt.Println("Error saving config:", err)
			return
		}

		fmt.Printf("Removed group %s from all repositories.\n", groupName)
	},
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

		names := uniqueGroupNames(cfg)
		if len(names) == 0 {
			fmt.Println("No groups configured.")
			return
		}

		for _, name := range names {
			fmt.Println(name)
		}
	},
}

func init() {
	groupCmd.AddCommand(addGroupCmd)
	groupCmd.AddCommand(removeGroupCmd)
	groupCmd.AddCommand(listGroupsCmd)
	rootCmd.AddCommand(groupCmd)
}
