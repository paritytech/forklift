package Commands

import (
	"forklift/Rpc"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start forklift coordinator server for current folder",
	PostRun: func(cmd *cobra.Command, args []string) {

		log.Error("olololololo")

	},
	Run: func(cmd *cobra.Command, args []string) {

		var rpcServer = Rpc.NewForkliftServer()
		rpcServer.Start()

	},
}
