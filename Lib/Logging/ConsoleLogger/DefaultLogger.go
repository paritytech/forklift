package ConsoleLogger

var defaultFormatter = TextFormatter{
	Indentation: 0,
}

var defaultLogger = Logger{
	Name:      "name",
	Level:     InfoLevel,
	Formatter: &defaultFormatter,
}

func SetLevel(level VerboseLevel) {
	defaultLogger.SetLevel(level)
}

func GetLevel() VerboseLevel {
	return defaultLogger.Level
}

func SetFormatter(formatter IFormatter) {
	defaultLogger.Formatter = formatter
}

func GetFormatter() IFormatter {
	return defaultLogger.Formatter
}

func Panicf(format string, v ...interface{}) {
	defaultLogger.Log(PanicLevel, nil, format, v...)
}

func Fatalf(format string, v ...interface{}) {
	defaultLogger.Log(FatalLevel, nil, format, v...)
}

func Errorf(format string, v ...interface{}) {
	defaultLogger.Log(ErrorLevel, nil, format, v...)
}

func Warningf(format string, v ...interface{}) {
	defaultLogger.Log(WarningLevel, nil, format, v...)
}

func Infof(format string, v ...interface{}) {
	defaultLogger.Log(InfoLevel, nil, format, v...)
}

func Debugf(format string, v ...interface{}) {
	defaultLogger.Log(DebugLevel, nil, format, v...)
}

func Tracef(format string, v ...interface{}) {
	defaultLogger.Log(TraceLevel, nil, format, v...)
}
