package Commands

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var Version = "0.2.0"

var rootCmd = &cobra.Command{
	Use:   "forklift <command> [flags] [cargo_project_dir]",
	Short: "Cargo cache management utility",
	Args:  cobra.MatchAll(),
	//DisableFlagParsing:    true,
	//DisableFlagsInUseLine: true,

	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("%s", args)
	},
}

func Execute() {

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
