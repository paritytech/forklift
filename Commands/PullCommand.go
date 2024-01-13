package Commands

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(pullCmd)
}

var pullCmd = &cobra.Command{
	Use:        "pull [flags] [project_dir]",
	Short:      "Download cache artifacts",
	Deprecated: "Use forklift as RUSTC_WRAPPER/cargo wrapper instead",
	Run: func(cmd *cobra.Command, args []string) {
	},
}
