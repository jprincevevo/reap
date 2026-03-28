package cmd

import (
	"fmt"
	"os"

	"github.com/jprincevevo/reap/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage reap configuration",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var configExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export the current config to stdout as YAML",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, _, err := config.Load()
		if err != nil {
			fmt.Println("Error loading config:", err)
			return
		}

		data, err := yaml.Marshal(cfg)
		if err != nil {
			fmt.Println("Error marshaling config:", err)
			return
		}

		fmt.Print(string(data))
	},
}

var configImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import repos and groups from a YAML file into the local config",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, err := os.ReadFile(args[0])
		if err != nil {
			fmt.Printf("Error reading file: %s\n", err)
			return
		}

		var imported config.Config
		if err := yaml.Unmarshal(data, &imported); err != nil {
			fmt.Printf("Error parsing YAML: %s\n", err)
			return
		}

		cfg, _, err := config.Load()
		if err != nil {
			fmt.Println("Error loading config:", err)
			return
		}

		repoCount := 0
		groupCount := 0
		for _, repo := range imported.Repos {
			if cfg.AddRepo(repo.URL) {
				repoCount++
			}
			for _, g := range repo.Groups {
				groupCount += cfg.ApplyGroupToRepos(g.Name, []string{repo.URL})
			}
		}

		if err := config.Save(cfg); err != nil {
			fmt.Println("Error saving config:", err)
			return
		}

		fmt.Printf("Imported %d repos, %d groups.\n", repoCount, groupCount)
	},
}

func init() {
	configCmd.AddCommand(configExportCmd)
	configCmd.AddCommand(configImportCmd)
	rootCmd.AddCommand(configCmd)
}
