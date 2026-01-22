package cmd

import (
	"fmt"
	"reap/config"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
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

func init() {
	rootCmd.AddCommand(addCmd)
}
