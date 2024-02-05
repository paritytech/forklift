package Config

import (
	"errors"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
)

var AppConfig = ForkliftConfig{
	Storage: ForkliftStorage{
		Type: "null",
	},
	Compression: ForkliftCompression{
		Type: "none",
	},
	General: ForkliftGeneral{
		LogLevel:     "info",
		ThreadsCount: 2,
	},
	Metrics: ForkliftMetrics{
		Enabled: false,
	},
}

func (c *ForkliftConfig) Get(key string) interface{} {
	return globalConfig.Get(key)
}

func (c *ForkliftConfig) GetInt(key string) int64 {
	return globalConfig.Int64(key)
}

func (c *ForkliftConfig) GetString(key string) string {
	return globalConfig.String(key)
}

func (c *ForkliftConfig) GetBool(key string) bool {
	return globalConfig.Bool(key)
}

func (c *ForkliftConfig) GetFloat(key string) float64 {
	return globalConfig.Float64(key)
}

func (c *ForkliftConfig) GetMap(key string) *map[string]interface{} {
	var resultMap = globalConfig.Get(key)

	if resultMap == nil {
		return &map[string]interface{}{}
	}

	var convertedMap = resultMap.(map[string]interface{})
	return &convertedMap
}

func (c *ForkliftConfig) GetAll() ([]byte, error) {
	marshalled, err := globalConfig.Marshal(toml.Parser())
	if err != nil {
		return nil, errors.Join(errors.New("unable to marshal config"), err)
	}

	return marshalled, nil
}

func (c *ForkliftConfig) Set(key string, value any) {
	err := localConfig.Set(key, value)
	if err != nil {
		return
	}
}

func (c *ForkliftConfig) Save() error {
	marshalled, err := localConfig.Marshal(toml.Parser())
	if err != nil {
		return errors.Join(errors.New("unable to marshal config"), err)
	}

	f, err := os.Create(".forklift/config.toml")
	if err != nil {
		return errors.Join(errors.New("unable to create config file"), err)
	}

	_, err = f.Write(marshalled)
	if err != nil {
		return errors.Join(errors.New("unable to write to config file"), err)
	}

	err = f.Close()
	if err != nil {
		return errors.Join(errors.New("unable to close config file"), err)
	}

	return nil
}

var (
	globalConfig = koanf.New(".")
	localConfig  = koanf.New(".")
)

func Init() error {

	var homeDir, _ = os.UserHomeDir()
	var path = filepath.Join(homeDir, ".forklift/config.toml")

	// home folder config
	errGlobal := globalConfig.Load(file.Provider(path), toml.Parser())
	if errGlobal != nil {
		log.Infof("unable to load global config: %v\n", errGlobal)
	}

	// current folder config
	if workloadDir, ok := os.LookupEnv("FORKLIFT_WORK_DIR"); ok {
		// FORKLIFT_WORK_DIR config
		path = filepath.Join(workloadDir, ".forklift/config.toml")
		_ = localConfig.Load(file.Provider(path), toml.Parser())
	} else {
		var pathLocal, _ = filepath.Abs(".forklift/config.toml")
		errLocal := localConfig.Load(file.Provider(pathLocal), toml.Parser())
		if errLocal != nil {
			log.Infof("unable to load local config: %v\n", errLocal)
		}

		if errGlobal != nil && errLocal != nil {
			return errors.Join(errors.New("unable to load config"), errGlobal, errLocal)
		}
	}

	err := globalConfig.Merge(localConfig)
	if err != nil {
		return errors.Join(errors.New("unable to merge configs"), err)
	}

	if err := globalConfig.Unmarshal("", &AppConfig); err != nil {
		return errors.Join(errors.New("unable to unmarshal config"), err)
	}

	// environment variables config
	err = globalConfig.Load(env.Provider("FORKLIFT_", ".", func(s string) string {
		return strings.Replace(
			strings.TrimPrefix(s, "FORKLIFT_"), "_", ".", -1)
	}), nil)
	if err != nil {
		return errors.Join(errors.New("unable to load env"), err)
	}

	return nil
}
