package cmd

import (
	"fmt"
	"os"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/spf13/cobra"
)

// diskCmd represents the disk command
var diskCmd = &cobra.Command{
	Use:   "disk",
	Short: "Check available disk space",
	Long:  `Display the available disk space and percentage for the root filesystem.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get disk usage for root filesystem
		usage, err := disk.Usage("/")
		if err != nil {
			fmt.Printf("Error getting disk usage: %v\n", err)
			os.Exit(1)
		}

		// Calculate available space percentage
		availablePercent := float64(usage.Free) / float64(usage.Total) * 100

		// Output the results
		fmt.Printf("Total space: %d bytes\n", usage.Total)
		fmt.Printf("Used space: %d bytes\n", usage.Used)
		fmt.Printf("Free space: %d bytes\n", usage.Free)
		fmt.Printf("Available space percentage: %.2f%%\n", availablePercent)
	},
}

func init() {
	rootCmd.AddCommand(diskCmd)
}