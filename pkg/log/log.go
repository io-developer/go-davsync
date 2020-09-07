package log

type Level uint

const (
	DebugLevel Level = iota + 1
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

var DefaultLogger *Logger = Newlogger()

func Debug(a ...interface{}) {
	DefaultLogger.Debug(a...)
}

func Debugf(format string, a ...interface{}) {
	DefaultLogger.Debugf(format, a...)
}

func Info(a ...interface{}) {
	DefaultLogger.Info(a...)
}

func Infof(format string, a ...interface{}) {
	DefaultLogger.Infof(format, a...)
}

func Warn(a ...interface{}) {
	DefaultLogger.Warn(a...)
}

func Warnf(format string, a ...interface{}) {
	DefaultLogger.Warnf(format, a...)
}

func Error(a ...interface{}) {
	DefaultLogger.Error(a...)
}

func Errorf(format string, a ...interface{}) {
	DefaultLogger.Errorf(format, a...)
}

func Fatal(a ...interface{}) {
	DefaultLogger.Fatal(a...)
}

func Fatalf(format string, a ...interface{}) {
	DefaultLogger.Fatalf(format, a...)
}
