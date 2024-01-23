package Lib

import log "github.com/sirupsen/logrus"

var AppConfig = ForkliftConfig{
	Storage:     ForkliftStorage{},
	Compression: ForkliftCompression{},
	General: ForkliftGeneral{
		Params:   map[string]string{},
		LogLevel: "",
	},
}

type ForkliftConfig struct {
	Storage     ForkliftStorage
	Compression ForkliftCompression
	General     ForkliftGeneral
}

type ForkliftStorage struct {
	Type string
}

type ForkliftCompression struct {
	Type string
}

type ForkliftGeneral struct {
	Params          map[string]string
	LogLevel        string
	LogrusLogLevel  log.Level
	ThreadsCount    int
	JobNameVariable string
	JobsBlacklist   []string
}
