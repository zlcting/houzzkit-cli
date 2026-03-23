package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// cleanlogsCmd represents the cleanlogs command
var cleanlogsCmd = &cobra.Command{
	Use:   "cleanlogs",
	Short: "Clear specific log files",
	Long:  `Truncate the contents of /var/log/houzzkit.log and /var/hass/config/home-assistant.log to clear their logs.`,
	Run: func(cmd *cobra.Command, args []string) {
		logFiles := []string{
			"/var/log/houzzkit.log",
			"/var/hass/config/home-assistant.log",
		}

		for _, file := range logFiles {
			err := os.Truncate(file, 0)
			if err != nil {
				fmt.Printf("Error clearing %s: %v\n", file, err)
			} else {
				fmt.Printf("Successfully cleared %s\n", file)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(cleanlogsCmd)
}
