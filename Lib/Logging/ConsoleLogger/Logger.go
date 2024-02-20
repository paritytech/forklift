package ConsoleLogger

import (
	"fmt"
)

type Logger struct {
	Name      string       // Name of the logger
	Level     VerboseLevel // Level of logging
	Formatter IFormatter   // IFormatter
	Fields    Fields       // Fields
}

func NewLogger(name string) *Logger {
	var l = &Logger{
		Name:      name,
		Level:     defaultLogger.Level,
		Formatter: &defaultFormatter,
	}
	return l
}

func NewLoggerWithFormatter(name string, formatter IFormatter) *Logger {
	var l = &Logger{
		Name:      name,
		Level:     defaultLogger.Level,
		Formatter: formatter,
	}
	return l
}

func (l *Logger) WithFields(fields Fields) *Logger {
	l.Fields = fields
	return l
}

func (l *Logger) Log(vLevel VerboseLevel, err error, format string, v ...interface{}) {

	if vLevel > l.Level {
		return
	}

	var message = fmt.Sprintf(format, v...)
	if err != nil {
		message = l.Formatter.FormatError(message, err)
	} else {
		message = l.Formatter.Format(message)
	}

	fmt.Printf("%s\n", message)
}

func (l *Logger) SetLevel(level VerboseLevel) {
	l.Level = level
}

func (l *Logger) Logf(vLevel VerboseLevel, format string, v ...interface{}) {
	l.Log(vLevel, nil, format, v...)
}

func (l *Logger) Panicf(format string, v ...interface{}) {
	l.Log(PanicLevel, nil, format, v...)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.Log(FatalLevel, nil, format, v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.Log(ErrorLevel, nil, format, v...)
}

func (l *Logger) Warningf(format string, v ...interface{}) {
	l.Log(WarningLevel, nil, format, v...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.Log(InfoLevel, nil, format, v...)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.Log(DebugLevel, nil, format, v...)
}

func (l *Logger) Tracef(format string, v ...interface{}) {
	l.Log(TraceLevel, nil, format, v...)
}
