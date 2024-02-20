package ConsoleLogger

import (
	"fmt"
	"regexp"
)

var newLineRegex = regexp.MustCompile(`\r?\n`)

type IFormatter interface {
	Format(message string) string
	FormatError(message string, err error) string
}

type TextFormatter struct {
	Indentation int // Indentation level
}

func (t *TextFormatter) Format(message string) string {
	//fmt.Println(t.Indentation)
	if t.Indentation > 0 {
		message = newLineRegex.ReplaceAllString(message, fmt.Sprintf("\n%*s", t.Indentation, ""))
		return fmt.Sprintf("%*s%s", t.Indentation, "", message)
	} else {
		return message
	}
}

func (t *TextFormatter) FormatError(message string, err error) string {
	var messageWithError = t.Format(fmt.Sprintf("%s\n%s", message, err.Error()))
	return t.Format(messageWithError)
}
