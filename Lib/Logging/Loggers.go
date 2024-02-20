package Logging

import (
	log "forklift/Lib/Logging/ConsoleLogger"
)

func CreateLogger(name string, indentation int, fields log.Fields) *log.Logger {

	var l = log.NewLoggerWithFormatter(name, &log.TextFormatter{Indentation: indentation})

	if fields == nil {
		fields = log.Fields{}
	}

	var logger = l.WithFields(fields)

	return logger
}
