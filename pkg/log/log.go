package log

import (
	"fmt"
	stdLog "log"
)

type Level uint

const (
	DebugLevel Level = iota + 1
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

var DefaultLogger *Logger = Newlogger()

type Logger struct {
	level Level
}

func Newlogger() *Logger {
	return &Logger{}
}

func (l *Logger) SetLevel(level Level) {
	l.level = level
}

func (l *Logger) Debug(a ...interface{}) {
	if l.level <= DebugLevel {
		fmt.Println(a...)
	}
}

func (l *Logger) Debugf(format string, a ...interface{}) {
	if l.level <= DebugLevel {
		fmt.Printf(format, a...)
	}
}

func (l *Logger) Info(a ...interface{}) {
	if l.level <= InfoLevel {
		fmt.Println(a...)
	}
}

func (l *Logger) Infof(format string, a ...interface{}) {
	if l.level <= InfoLevel {
		fmt.Printf(format, a...)
	}
}

func (l *Logger) Warn(a ...interface{}) {
	if l.level <= WarnLevel {
		stdLog.Println(a...)
	}
}

func (l *Logger) Warnf(format string, a ...interface{}) {
	if l.level <= WarnLevel {
		stdLog.Printf(format, a...)
	}
}

func (l *Logger) Error(a ...interface{}) {
	if l.level <= ErrorLevel {
		stdLog.Println(a...)
	}
}

func (l *Logger) Errorf(format string, a ...interface{}) {
	if l.level <= ErrorLevel {
		stdLog.Printf(format, a...)
	}
}

func (l *Logger) Fatal(a ...interface{}) {
	if l.level <= FatalLevel {
		stdLog.Fatalln(a...)
	}
}

func (l *Logger) Fatalf(format string, a ...interface{}) {
	if l.level <= FatalLevel {
		stdLog.Fatalf(format, a...)
	}
}

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
