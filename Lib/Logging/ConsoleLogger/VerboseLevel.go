package ConsoleLogger

type VerboseLevel uint32

const (
	PanicLevel   VerboseLevel = 0
	FatalLevel   VerboseLevel = 2
	ErrorLevel   VerboseLevel = 4
	WarningLevel VerboseLevel = 6
	InfoLevel    VerboseLevel = 8
	DebugLevel   VerboseLevel = 10
	TraceLevel   VerboseLevel = 12
)

func ParseLevel(level string) (VerboseLevel, error) {
	switch level {
	case "panic":
		return PanicLevel, nil
	case "fatal":
		return FatalLevel, nil
	case "error":
		return ErrorLevel, nil
	case "warn":
	case "warning":
		return WarningLevel, nil
	case "info":
		return InfoLevel, nil
	case "debug":
		return DebugLevel, nil
	case "trace":
		return TraceLevel, nil
	}
	return InfoLevel, nil
}
