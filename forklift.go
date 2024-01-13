package main

import (
	"forklift/Commands"
	"forklift/Commands/Server"
	"forklift/Commands/Wrapper"
	"forklift/Lib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"path"
	"strings"
)

func main() {

	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("toml")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".forklift")
	viper.AddConfigPath("$HOME/.forklift")
	if wd, ok := os.LookupEnv("FORKLIFT_WORK_DIR"); ok {
		viper.AddConfigPath(path.Join(wd, ".forklift"))
	}

	//viper.AddConfigPath("$HOME/.forklift") // call multiple times to add many search paths

	viper.SetDefault("storage.type", "s3")
	viper.SetDefault("compression.type", "zstd")
	viper.SetDefault("general.params", map[string]string{})
	viper.SetDefault("general.threadsCount", 0)

	log.SetFormatter(&log.TextFormatter{})

	var err = viper.ReadInConfig()
	if err != nil {
		log.Errorln(err)
	}

	err = viper.Unmarshal(&Lib.AppConfig)
	if err != nil {
		log.Errorln(err)
	}

	logLevel, err := log.ParseLevel(Lib.AppConfig.General.LogLevel)
	if err != nil {
		logLevel = log.InfoLevel
		log.Debugf("unknown log level (verbose) `%s`, using default `info`\n", Lib.AppConfig.General.LogLevel)
	}
	log.SetLevel(logLevel)

	//log.Debugf("Checking for blacklisted jobs")

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
