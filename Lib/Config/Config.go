package Config

import (
	"errors"
	log "forklift/Lib/Logging/ConsoleLogger"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"os"
	"path/filepath"
	"strings"
)

var AppConfig = ForkliftConfig{
	Storage: ForkliftStorage{
		Type:     "null",
		ReadOnly: false,
	},
	Compression: ForkliftCompression{
		Type: "none",
	},
	General: ForkliftGeneral{
		LogLevel:     "info",
		ThreadsCount: 2,

		Quiet: false,
	},
	Metrics: ForkliftMetrics{
		Enabled:     false,
		ExtraLabels: map[string]string{},
	},
	Cache: ForkliftCache{
		ExtraEnv:      []string{},
		ExtraMetadata: map[string]string{},
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

func (c *ForkliftConfig) Set(key string, value any) error {
	err := localConfig.Set(key, value)
	if err != nil {
		return errors.Join(errors.New("unable to set config"), err)
	}
	globalConfig.Merge(localConfig)
	return nil
}

func (c *ForkliftConfig) Delete(key string) {
	localConfig.Delete(key)
}

func (c *ForkliftConfig) Save(configPath *string) error {
	marshalled, err := localConfig.Marshal(toml.Parser())
	if err != nil {
		return errors.Join(errors.New("unable to marshal config"), err)
	}

	var localPath string

	if configPath == nil {
		_ = os.Mkdir(".forklift", 0755)
		localPath = ".forklift/config.toml"
	} else {
		localPath = *configPath
	}

	f, err := os.Create(localPath)
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

func Init(configPath *string) error {

	var homeDir, _ = os.UserHomeDir()
	var path = filepath.Join(homeDir, ".forklift/config.toml")

	// home folder config
	errGlobal := globalConfig.Load(file.Provider(path), toml.Parser())
	if errGlobal != nil {
		log.Tracef("unable to load global config: %v\n", errGlobal)
	}

	// current folder config
	if configPath == nil {

		if workloadDir, ok := os.LookupEnv("FORKLIFT_WORK_DIR"); ok {
			// FORKLIFT_WORK_DIR config
			path = filepath.Join(workloadDir, ".forklift/config.toml")
			_ = localConfig.Load(file.Provider(path), toml.Parser())
		} else {
			var pathLocal, _ = filepath.Abs(".forklift/config.toml")
			errLocal := localConfig.Load(file.Provider(pathLocal), toml.Parser())
			if errLocal != nil {
				log.Tracef("unable to load local config: %v\n", errLocal)
			}

			if errGlobal != nil && errLocal != nil {
				return errors.Join(errors.New("unable to load config"), errGlobal, errLocal)
			}
		}
	} else {
		// custom config
		errCustom := localConfig.Load(file.Provider(*configPath), toml.Parser())
		if errCustom != nil {
			return errors.Join(errors.New("unable to load custom config"), errCustom)
		}
	}

	err := globalConfig.Merge(localConfig)
	if err != nil {
		return errors.Join(errors.New("unable to merge configs"), err)
	}

	// environment variables config
	err = globalConfig.Load(env.Provider("FORKLIFT_", ".", func(s string) string {
		return strings.Replace(
			strings.TrimPrefix(s, "FORKLIFT_"), "_", ".", -1)
	}), nil)
	/*if err != nil {
		return errors.Join(errors.New("unable to load env"), err)
	}*/

	if err := globalConfig.Unmarshal("", &AppConfig); err != nil {
		return errors.Join(errors.New("unable to unmarshal config"), err)
	}

	return nil
}
