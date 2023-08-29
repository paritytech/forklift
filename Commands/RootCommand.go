package Commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var storage string
var compression string
var mode string
var params map[string]string

var rootCmd = &cobra.Command{
	Use:     "forklift <command> [flags] [cargo_project_dir]",
	Short:   "Cargo cache management utility",
	Args:    cobra.MaximumNArgs(1),
	Version: "0.1.1",
}

func Execute() {

	rootCmd.PersistentFlags().StringVarP(&storage, "storage", "s", "s3", "Storage driver\nAvailable: s3")
	rootCmd.PersistentFlags().StringVarP(&compression, "compression", "c", "none", "Compression algorithm to use\nAvailable: none, xz")
	rootCmd.PersistentFlags().StringToStringVarP(&params, "param", "p", nil, "map of additional parameters\n ex: -p S3_BUCKET_NAME=my_bucket")
	rootCmd.PersistentFlags().StringVarP(&mode, "mode", "m", "debug", "Available: debug, release")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
