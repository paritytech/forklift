package Commands

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(stopCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop forklift coordinator server for current location",
	Run: func(cmd *cobra.Command, args []string) {

		/*
			var client = Rpc.NewForkliftControlClient()

			err := client.Connect()
			if err != nil {
				log.Fatal(err)
				return
			}

			client.Stop()
		*/
	},
}
