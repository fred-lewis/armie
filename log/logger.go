package log

import (
	"io"
	"time"
	"fmt"
	"os"
)

type LogLevel int
const (
	TRACE LogLevel = iota + 1
	DEBUG
	INFO
	WARNING
	ERROR
)

const tfmt = "2006-01-02 15:04:05.999 MST"

type Logger struct {
	out io.Writer
	level LogLevel
}

func New(out io.Writer) *Logger {
	return &Logger{
		out: out,
		level: INFO,
	}
}

func (l *Logger) Log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	tstr := time.Now().Format(tfmt)

	pfxs := []string{"[TRACE]", "[DEBUG]", "[INFO]", "[WARN]", "[ERROR]"}

	format = tstr + " " + pfxs[level - 1] + " " + format + "\n"

	if len(args) > 0 {
		fmt.Printf(format, args...)
	} else {
		fmt.Print(format)
	}
	os.Stdout.Sync()
}

func (l *Logger) Trace(format string, args... interface{}) {
	l.Log(TRACE, format, args...)
}

func (l *Logger) Debug(format string, args... interface{}) {
	l.Log(DEBUG, format, args...)
}

func (l *Logger) Info(format string, args... interface{}) {
	l.Log(INFO, format, args...)
}

func (l *Logger) Warn(format string, args... interface{}) {
	l.Log(WARNING, format, args...)
}

func (l *Logger) Error(format string, args... interface{}) {
	l.Log(ERROR, format, args...)
}

func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

func (l *Logger) GetWriter() io.Writer {
	return l.out
}