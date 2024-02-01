package Config

import (
	"errors"
	"github.com/spf13/viper"
	"os"
	"path"
)

var AppConfig = ForkliftConfig{
	Storage:     ForkliftStorage{},
	Compression: ForkliftCompression{},
	General: ForkliftGeneral{
		Params:   map[string]string{},
		LogLevel: "",
	},
	Metrics: ForkliftMetrics{},
}

func ViperInit() error {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("toml")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".forklift")
	viper.AddConfigPath("$HOME/.forklift")
	if wd, ok := os.LookupEnv("FORKLIFT_WORK_DIR"); ok {
		viper.AddConfigPath(path.Join(wd, ".forklift"))
	}

	viper.SetDefault("storage.type", "null")
	viper.SetDefault("compression.type", "none")
	viper.SetDefault("general.params", map[string]string{})
	viper.SetDefault("general.threadsCount", 0)
	viper.SetDefault("metrics.enabled", false)
	viper.SetDefault("metrics.extraLabels", map[string]string{})

	var err = viper.ReadInConfig()
	if err != nil {
		return errors.Join(errors.New("unable to read config"), err)
	}

	err = viper.Unmarshal(&AppConfig)
	if err != nil {
		return errors.Join(errors.New("unable to unmarshal config"), err)
	}

	return nil
}
