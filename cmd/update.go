package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/creativeprojects/go-selfupdate"
	"github.com/jprincevevo/reap/version"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update reap to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		updater, err := selfupdate.NewUpdater(selfupdate.Config{})
		if err != nil {
			fmt.Println("Error creating updater:", err)
			return
		}

		repo := selfupdate.ParseSlug("jprincevevo/reap")

		latest, found, err := updater.DetectLatest(context.Background(), repo)
		if err != nil {
			fmt.Println("Error getting latest release:", err)
			return
		}

		if !found || latest == nil {
			fmt.Println("No release found")
			return
		}

		if latest.Version() == version.Version {
			fmt.Println("You are already using the latest version of reap")
			return
		}

		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingTop(1).
			PaddingBottom(1).
			PaddingLeft(2).
			PaddingRight(2)

		fmt.Println(style.Render(fmt.Sprintf("A new version of reap is available: %s", latest.Version())))

		fmt.Print("Do you want to update? (y/n) ")
		var response string
		fmt.Scanln(&response)

		if response != "y" {
			fmt.Println("Update aborted")
			return
		}

		exePath, err := os.Executable()
		if err != nil {
			fmt.Println("Error getting executable path:", err)
			return
		}

		err = updater.UpdateTo(context.Background(), latest, exePath)
		if err != nil {
			fmt.Println("Error updating:", err)
			return
		}

		fmt.Println("Successfully updated to version", latest.Version())
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
