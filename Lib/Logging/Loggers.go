package Logging

import (
	log "github.com/sirupsen/logrus"
	"os"
)

func CreateLogger(name string, indentation int, fields log.Fields) *log.Entry {

	var l = log.Logger{
		Out:       os.Stderr,
		Formatter: &ForkliftTextFormatter{Indentation: indentation, TaskPrefix: name},
		Level:     log.GetLevel(),
	}

	if fields == nil {
		fields = log.Fields{}
	}

	var logger = l.WithFields(fields)

	return logger
}
