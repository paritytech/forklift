package main

import (
	"forklift/Commands"
	"forklift/Commands/Server"
	"forklift/Commands/Wrapper"
	"forklift/Lib/Config"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

func main() {

	log.SetFormatter(&log.TextFormatter{})

	var err = Config.ViperInit()
	if err != nil {
		log.Errorln(err)
		log.Infof("Config error, bypassing forklift: %s", err)
		Server.BypassForklift()
		return
	}

	logLevel, err := log.ParseLevel(Config.AppConfig.General.LogLevel)
	if err != nil {
		logLevel = log.InfoLevel
		log.Debugf("unknown log level (verbose) `%s`, using default `info`\n", Config.AppConfig.General.LogLevel)
	}
	log.SetLevel(logLevel)

	if len(os.Args) > 1 &&
		(strings.Contains(os.Args[1], "rustc") || strings.Contains(os.Args[1], "clippy-driver")) {

		Wrapper.Run(os.Args[1:])
	} else if len(os.Args) > 1 && strings.Contains(os.Args[1], "cargo") {
		Server.Run(os.Args[1:])
	} else {
		Commands.Execute()
	}

	return
}
