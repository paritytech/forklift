package Commands

import (
	"forklift/Commands/Server"
	"forklift/Commands/Wrapper"
	"forklift/Lib/Config"
	"forklift/Lib/Logging"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

var Version = "0.10.0"

var logger = Logging.CreateLogger("forklift", 2, nil)

var rootCmd = &cobra.Command{
	Use:                   "forklift <something>",
	Short:                 "Cargo cache management utility",
	Example:               "forklift cargo build ...\nforklift rustc ...\nforklift config set storage.type s3",
	Args:                  cobra.MatchAll(),
	DisableFlagParsing:    true,
	DisableFlagsInUseLine: true,
	Version:               Version,
	Run:                   rootRun,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Errorln(err)
		os.Exit(1)
	}
}

func rootRun(cmd *cobra.Command, args []string) {

	if bypass, ok := os.LookupEnv("FORKLIFT_BYPASS"); ok && bypass == "true" {
		logger.Infof("FORKLIFT_BYPASS is set to 'true', bypassing forklift")
		Server.BypassForklift()
		return
	}

	var err = Config.Init()
	if err != nil {
		logger.Errorf("Config error, bypassing: %s", err)
		Server.BypassForklift()
		return
	}

	logLevel, err := log.ParseLevel(Config.AppConfig.General.LogLevel)
	if err != nil {
		logLevel = log.InfoLevel
		logger.Errorf("unknown log level `%s`, using default `info`", Config.AppConfig.General.LogLevel)
	}
	log.SetLevel(logLevel)

	if len(args) == 0 {
		cmd.Help()
	} else if strings.Contains(os.Args[1], "rustc") || strings.Contains(os.Args[1], "clippy-driver") {
		Wrapper.Run(args)
	} else if strings.Contains(os.Args[1], "cargo") {
		Server.Run(args)
	} else {
		Server.BypassForklift()
	}
}
