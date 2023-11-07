package Commands

import (
	"forklift/Rpc"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start forklift coordinator server for current folder",
	Run: func(cmd *cobra.Command, args []string) {

		var rpcServer = Rpc.NewForkliftServer()
		rpcServer.Start()

	},
}
