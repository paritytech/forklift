package Commands

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
)

var Version = "0.2.0"

var storage string
var compression string
var mode string
var params map[string]string
var verboseLevel string
var extraDirs []string

var rootCmd = &cobra.Command{
	Use:     "forklift <command> [flags] [cargo_project_dir]",
	Short:   "Cargo cache management utility",
	Args:    cobra.MaximumNArgs(1),
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var logLevel, err = log.ParseLevel(verboseLevel)
		if err != nil {
			logLevel = log.InfoLevel
			log.Infof("unknown verbose level `%s`, using default `info`\n", verboseLevel)
		}

		log.SetLevel(logLevel)
	},
}

func Execute() {

	rootCmd.PersistentFlags().StringVarP(&storage, "storage", "s", "s3", "Storage driver\nAvailable: s3, fs")
	rootCmd.PersistentFlags().StringVarP(&compression, "compression", "c", "none", "Compression algorithm to use\nAvailable: none, xz")
	rootCmd.PersistentFlags().StringToStringVarP(&params, "param", "p", nil, "map of additional parameters\n ex: -p S3_BUCKET_NAME=my_bucket")
	rootCmd.PersistentFlags().StringVarP(&mode, "mode", "m", "debug", "Available: debug, release")
	rootCmd.PersistentFlags().StringVarP(&verboseLevel, "verbose", "v", "info", "Available: panic, fatal, error, warn, warning, info, debug, trace")
	rootCmd.PersistentFlags().StringArrayVar(&extraDirs, "extra-dir", nil, "")

	if err := rootCmd.Execute(); err != nil {
		log.Errorln(err)
		os.Exit(1)
	}
}
