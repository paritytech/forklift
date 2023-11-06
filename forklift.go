package main

import (
	"forklift/Commands"
	"forklift/Commands/Server"
	"forklift/Commands/Wrapper"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"path"
	"strings"
)

func main() {

	viper.SetConfigName("config")    // name of config file (without extension)
	viper.SetConfigType("toml")      // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".forklift") // path to look for the config file in
	if wd, ok := os.LookupEnv("FORKLIFT_WORK_DIR"); ok {
		viper.AddConfigPath(path.Join(wd, ".forklift"))
	}

	//viper.AddConfigPath("$HOME/.forklift") // call multiple times to add many search paths

	viper.SetDefault("storage.type", "s3")
	viper.SetDefault("compression.type", "zstd")
	viper.SetDefault("general.params", map[string]string{})

	log.SetFormatter(&log.TextFormatter{})

	var err = viper.ReadInConfig()
	if err != nil {
		log.Errorln(err)
	}

	if strings.Contains(os.Args[1], "rustc") {
		Wrapper.Run(os.Args[1:])
	} else if strings.Contains(os.Args[1], "cargo") {
		Server.Run(os.Args[1:])
	} else {
		Commands.Execute()
	}

	return
}
