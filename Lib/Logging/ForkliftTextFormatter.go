package Logging

/*
import (
	"fmt"
	log "forklift/Lib/Logging/ConsoleLogger"
	"strings"
)

type ForkliftTextFormatter struct {
	Indentation int
	TaskPrefix  string
}

func (f *ForkliftTextFormatter) Format(entry *log.Logger) ([]byte, error) {

	var sb = strings.Builder{}

	for name, value := range entry.Data {
		sb.WriteString(fmt.Sprintf(" %s: '%s'", name, value))
	}

	var logString = ""

	if len(entry.Data) > 0 {
		logString = fmt.Sprintf(
			"%*s%s %s: %s,%s\n",
			f.Indentation,
			"",
			f.TaskPrefix,
			entry.Level.String(),
			entry.Message,
			sb.String(),
		)
	} else {
		logString = fmt.Sprintf(
			"%*s%s %s: %s\n",
			f.Indentation,
			"",
			f.TaskPrefix,
			entry.Level.String(),
			entry.Message,
		)
	}

	return []byte(logString), nil
}
*/
