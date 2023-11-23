package Commands

import (
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path/filepath"
)

func init() {
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start detached forklift coordinator server for current location",
	Run: func(cmd *cobra.Command, args []string) {

		var execPath, _ = os.Executable()
		execPath, _ = filepath.EvalSymlinks(execPath)

		command := exec.Command(execPath, "serve")

		command.Start()
		command.Process.Release()

	},
}
