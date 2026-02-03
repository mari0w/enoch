package logging

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"enoch/internal/config"
)

type Level int

const (
	Debug Level = iota
	Info
	Warn
	Error
)

type Logger struct {
	level      Level
	console    bool
	color      bool
	timeFormat string
	consoleOut io.Writer
	fileOut    io.Writer
	file       *os.File
	mu         sync.Mutex
}

func New(cfg config.Config) (*Logger, error) {
	lvl, err := parseLevel(cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	var file *os.File
	var fileOut io.Writer
	if cfg.LogFile != "" {
		opened, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		file = opened
		fileOut = opened
	}

	logger := &Logger{
		level:      lvl,
		console:    cfg.LogConsole,
		color:      cfg.LogColor,
		timeFormat: cfg.LogTimeFormat,
		consoleOut: os.Stdout,
		fileOut:    fileOut,
		file:       file,
	}
	return logger, nil
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(Debug, format, args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(Info, format, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(Warn, format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(Error, format, args...)
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if l == nil {
		return
	}
	if level < l.level {
		return
	}

	timestamp := time.Now().Format(l.timeFormat)
	message := fmt.Sprintf(format, args...)
	levelText := level.String()

	line := fmt.Sprintf("%s [%s] %s", timestamp, levelText, message)

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.console {
		if l.color {
			colored := fmt.Sprintf("%s [%s%s%s] %s", timestamp, level.colorCode(), levelText, colorReset, message)
			_, _ = fmt.Fprintln(l.consoleOut, colored)
		} else {
			_, _ = fmt.Fprintln(l.consoleOut, line)
		}
	}

	if l.fileOut != nil {
		_, _ = fmt.Fprintln(l.fileOut, line)
	}
}

func parseLevel(level string) (Level, error) {
	switch level {
	case "debug":
		return Debug, nil
	case "info":
		return Info, nil
	case "warn":
		return Warn, nil
	case "error":
		return Error, nil
	default:
		return Info, fmt.Errorf("unknown log level: %s", level)
	}
}

func (l Level) String() string {
	switch l {
	case Debug:
		return "DEBUG"
	case Info:
		return "INFO"
	case Warn:
		return "WARN"
	case Error:
		return "ERROR"
	default:
		return "INFO"
	}
}

func (l Level) colorCode() string {
	switch l {
	case Debug:
		return "\033[36m"
	case Info:
		return "\033[32m"
	case Warn:
		return "\033[33m"
	case Error:
		return "\033[31m"
	default:
		return "\033[0m"
	}
}

const colorReset = "\033[0m"
