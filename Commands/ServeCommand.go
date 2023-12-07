package Commands

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run forklift coordinator server for current location",
	Run: func(cmd *cobra.Command, args []string) {
		/*var rpcServer = Rpc.NewForkliftServer()
		rpcServer.Start(".", )
		*/

	},
}
