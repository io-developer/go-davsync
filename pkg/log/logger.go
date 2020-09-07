package log

import (
	"fmt"
	stdLog "log"
)

type Logger struct {
	level Level
}

func Newlogger() *Logger {
	return &Logger{}
}

func (l *Logger) SetLevel(level Level) {
	l.level = level
}

func (l *Logger) Log(level Level, a ...interface{}) {
	if l.level > level {
		return
	}
	switch level {
	case DebugLevel:
		fmt.Println(a...)
		break
	case InfoLevel:
		fmt.Println(a...)
		break
	case WarnLevel:
		stdLog.Println(a...)
		break
	case ErrorLevel:
		stdLog.Println(a...)
		break
	case FatalLevel:
		stdLog.Fatalln(a...)
		break
	default:
		stdLog.Fatalln("Unknown level", level, a)
		break
	}
}

func (l *Logger) Logf(level Level, format string, a ...interface{}) {
	if l.level > level {
		return
	}
	switch level {
	case DebugLevel:
		fmt.Printf(format, a...)
		break
	case InfoLevel:
		fmt.Printf(format, a...)
		break
	case WarnLevel:
		stdLog.Printf(format, a...)
		break
	case ErrorLevel:
		stdLog.Printf(format, a...)
		break
	case FatalLevel:
		stdLog.Fatalf(format, a...)
		break
	default:
		stdLog.Fatalln("Unknown level", level, format, a)
		break
	}
}

func (l *Logger) Debug(a ...interface{}) {
	l.Log(DebugLevel, a...)
}

func (l *Logger) Debugf(format string, a ...interface{}) {
	l.Logf(DebugLevel, format, a...)
}

func (l *Logger) Info(a ...interface{}) {
	l.Log(InfoLevel, a...)
}

func (l *Logger) Infof(format string, a ...interface{}) {
	l.Logf(InfoLevel, format, a...)
}

func (l *Logger) Warn(a ...interface{}) {
	l.Log(WarnLevel, a...)
}

func (l *Logger) Warnf(format string, a ...interface{}) {
	l.Logf(WarnLevel, format, a...)
}

func (l *Logger) Error(a ...interface{}) {
	l.Log(ErrorLevel, a...)
}

func (l *Logger) Errorf(format string, a ...interface{}) {
	l.Logf(ErrorLevel, format, a...)
}

func (l *Logger) Fatal(a ...interface{}) {
	l.Log(FatalLevel, a...)
}

func (l *Logger) Fatalf(format string, a ...interface{}) {
	l.Logf(FatalLevel, format, a...)
}

func Log(level Level, a ...interface{}) {
	DefaultLogger.Log(level, a...)
}

func Logf(level Level, format string, a ...interface{}) {
	DefaultLogger.Logf(level, format, a...)
}
