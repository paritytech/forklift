package Commands

import (
	"github.com/spf13/cobra"
	"os"
	"path"
	"path/filepath"
)

func init() {
	rootCmd.AddCommand(cleanCmd)
}

var cleanCmd = &cobra.Command{
	Use:   "clean [project_dir]",
	Short: "Cleanup generated `items-cache` files",
	Run: func(cmd *cobra.Command, args []string) {
		var files, _ = filepath.Glob(path.Join(".forklift", "items-cache", "item-*"))

		for _, file := range files {
			os.Remove(file)
		}

		/*
			files, _ = filepath.Glob(path.Join("target", config.General.Dir, "forklift", "*"))
			for _, file := range files {
				os.Remove(file)
			}*/
	},
}
