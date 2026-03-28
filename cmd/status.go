package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/jprincevevo/reap/config"
	"github.com/spf13/cobra"
)

var (
	statusPresentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#2EC4B6"))
	statusMissingStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show which configured repos are present locally",
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

		for _, repo := range cfg.Repos {
			repoName := strings.TrimSuffix(filepath.Base(repo.URL), ".git")
			if _, err := os.Stat(repoName); err == nil {
				fmt.Printf("%s  %s      (present)\n",
					statusPresentStyle.Render("✓"), repoName)
			} else {
				fmt.Printf("%s  %s      (missing)\n",
					statusMissingStyle.Render("✗"), repoName)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
