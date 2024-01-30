package Commands

import (
	"forklift/Lib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var Version = "0.2.0"

//var storage string
//var compression string
//var mode string
//var params map[string]string
//var verboseLevel string
//var extraDirs []string

var rootCmd = &cobra.Command{
	Use:     "forklift <command> [flags] [cargo_project_dir]",
	Short:   "Cargo cache management utility",
	Args:    cobra.MaximumNArgs(1),
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var logLevel, err = log.ParseLevel(Lib.AppConfig.General.LogLevel)
		if err != nil {
			logLevel = log.InfoLevel

			log.Infof("unknown verbose level `%s`, using default `info`\n", Lib.AppConfig.General.LogLevel)
		}

		log.SetLevel(logLevel)

		mergeParams(&Lib.AppConfig.General.Params)
	},
}

func Execute() {

	/*
		rootCmd.PersistentFlags().StringVarP(&Lib.AppConfig.Storage.Type, "storage", "s", "s3", "Storage driver\nAvailable: s3, fs")
		viper.BindPFlag("storage.type", rootCmd.PersistentFlags().Lookup("storage"))

		rootCmd.PersistentFlags().StringVarP(&Lib.AppConfig.Compression.Type, "compression", "c", "zstd", "Compression algorithm to use\nAvailable: none, xz")
		viper.BindPFlag("compression.type", rootCmd.PersistentFlags().Lookup("compression"))

		rootCmd.PersistentFlags().StringVarP(&Lib.AppConfig.General.LogLevel, "verbose", "v", "info", "Available: panic, fatal, error, warn, warning, info, debug, trace")
		viper.BindPFlag("general.logLevel", rootCmd.PersistentFlags().Lookup("verbose"))

		rootCmd.PersistentFlags().StringToStringVarP(&Lib.AppConfig.General.Params, "param", "p", map[string]string{}, "map of additional parameters\n ex: -p S3_BUCKET_NAME=my_bucket")
	*/

	if err := rootCmd.Execute(); err != nil {
		log.Errorln(err)
		os.Exit(1)
	}
}

func mergeParams(params *map[string]string) {
	/*var viperParams = viper.GetStringMapString("storage.params")
	appendToMap(params, &viperParams)

	viperParams = viper.GetStringMapString("compression.params")
	appendToMap(params, &viperParams)*/

	var viperParams = viper.GetStringMapString("general.params")
	appendToMap(params, &viperParams)
}

func appendToMap(to *map[string]string, from *map[string]string) {
	for key, value := range *from {
		if _, b := (*to)[key]; !b {
			(*to)[key] = value
		}
	}
}
