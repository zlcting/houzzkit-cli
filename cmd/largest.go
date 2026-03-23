package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
)

type FileInfo struct {
	Path string
	Size int64
}

// formatBytes converts bytes to human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// largestCmd represents the largest command ./houzzkit_tool largest --count 5
var largestCmd = &cobra.Command{
	Use:   "largest",
	Short: "List the largest files on the host",
	Long:  `Traverse the filesystem and list the top N largest files.`,
	Run: func(cmd *cobra.Command, args []string) {
		count, _ := cmd.Flags().GetInt("count")

		var files []FileInfo

		// Walk the filesystem starting from root
		err := filepath.Walk("/", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// Skip directories/files we can't access
				return nil
			}
			if !info.IsDir() {
				files = append(files, FileInfo{Path: path, Size: info.Size()})
			}
			return nil
		})
		if err != nil {
			fmt.Printf("Error walking filesystem: %v\n", err)
			os.Exit(1)
		}

		// Sort files by size descending
		sort.Slice(files, func(i, j int) bool {
			return files[i].Size > files[j].Size
		})

		// Output the top N files
		for i := 0; i < len(files) && i < count; i++ {
			fmt.Printf("%s: %s\n", formatBytes(files[i].Size), files[i].Path)
		}
	},
}

func init() {
	rootCmd.AddCommand(largestCmd)
	largestCmd.Flags().IntP("count", "n", 10, "Number of largest files to list")
}
