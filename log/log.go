package log

import (
	gofmt "fmt"
	"io"
	golog "log"
	"os"
	"strings"
	"sync/atomic"
)

var logger *golog.Logger
var level Level = LevelInfo

type Level int64

const (
	LevelFatal Level = iota
	LevelError
	LevelWarning
	LevelInfo
	LevelDebug
)

func (l Level) String() string {
	switch l {
	case LevelFatal:
		return "FATAL"
	case LevelError:
		return "Error"
	case LevelWarning:
		return "WARNING"
	case LevelInfo:
		return "INFO"
	case LevelDebug:
		return "DEBUG"
	}
	return "Unknown-Level"
}

func Init(path string) error {
	var writer io.Writer
	if path == "" {
		gofmt.Println("log to console")
		writer = os.Stdout
	} else {
		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			gofmt.Printf("failed to open %s, %v\n", path, err)
			os.Exit(1)
		}
		writer = f
	}

	logger = golog.New(writer, "", golog.LstdFlags|golog.Lshortfile)
	return nil
}

func SetLevel(l Level) {
	atomic.StoreInt64((*int64)(&level), int64(l))
}

func SetLevelString(l string) {
	var level Level
	switch strings.ToLower(l) {
	case "info":
		level = LevelInfo
	case "debug":
		level = LevelDebug
	case "warn", "warning":
		level = LevelWarning
	case "error", "err":
		level = LevelError
	default:
		level = LevelInfo
	}
	SetLevel(level)
}

func leveledLog(l Level, fmt string, values ...interface{}) {
	c := Level(atomic.LoadInt64((*int64)(&level)))
	if l > c {
		return
	}
	fmt = "[%s] " + fmt
	newValues := make([]interface{}, len(values)+1)
	newValues[0] = l
	copy(newValues[1:], values)
	logger.Output(3, gofmt.Sprintf(fmt, newValues...))
}

func Info(fmt string, values ...interface{}) {
	leveledLog(LevelInfo, fmt, values...)
}

func Debug(fmt string, values ...interface{}) {
	leveledLog(LevelDebug, fmt, values...)
}

func Error(fmt string, values ...interface{}) {
	leveledLog(LevelError, fmt, values...)
}

func Warn(fmt string, values ...interface{}) {
	leveledLog(LevelWarning, fmt, values...)
}

func Fatal(fmt string, values ...interface{}) {
	leveledLog(LevelInfo, fmt, values...)
	panic(gofmt.Sprintf(fmt, values...))
}
