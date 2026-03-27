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
		cmd.Help()
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

		for i, repo := range cfg.Repos {
			for _, selected := range selectedRepos {
				if repo.URL == selected {
					alreadyHas := false
					for _, g := range cfg.Repos[i].Groups {
						if g.Name == groupName {
							alreadyHas = true
							break
						}
					}
					if !alreadyHas {
						cfg.Repos[i].Groups = append(cfg.Repos[i].Groups, config.Group{
							Name:     groupName,
							Selected: true,
						})
					}
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

		removed := 0
		for i, repo := range cfg.Repos {
			var newGroups []config.Group
			for _, group := range repo.Groups {
				if group.Name != groupName {
					newGroups = append(newGroups, group)
				} else {
					removed++
				}
			}
			cfg.Repos[i].Groups = newGroups
		}

		if removed == 0 {
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

		seenGroups := make(map[string]bool)
		for _, repo := range cfg.Repos {
			for _, group := range repo.Groups {
				if !seenGroups[group.Name] {
					seenGroups[group.Name] = true
					fmt.Println(group.Name)
				}
			}
		}

		if len(seenGroups) == 0 {
			fmt.Println("No groups configured.")
		}
	},
}

func init() {
	groupCmd.AddCommand(addGroupCmd)
	groupCmd.AddCommand(removeGroupCmd)
	groupCmd.AddCommand(listGroupsCmd)
	rootCmd.AddCommand(groupCmd)
}
