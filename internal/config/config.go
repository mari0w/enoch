package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	TelegramBotToken       string
	TelegramAllowedChatID  string
	TelegramPollInterval   time.Duration
	TelegramTypingInterval time.Duration
	TelegramContextSize    int
	CodexCommand           string
	CodexArgs              []string
	CodexPromptMode        string
	CodexTimeout           time.Duration
	CodexWorkdir           string
	CodexDisableCPR        bool
	CodexUseTTY            bool
	CodexProgressInterval  time.Duration
	LogLevel               string
	LogFile                string
	LogConsole             bool
	LogColor               bool
	LogTimeFormat          string
}

func Load() (Config, error) {
	_ = LoadDotEnv(".env")

	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		return Config{}, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}

	allowedChat := strings.TrimSpace(os.Getenv("TELEGRAM_ALLOWED_CHAT_ID"))

	pollInterval := 2 * time.Second
	pollRaw := strings.TrimSpace(os.Getenv("TELEGRAM_POLL_INTERVAL"))
	if pollRaw != "" {
		seconds, err := strconv.ParseFloat(pollRaw, 64)
		if err == nil && seconds > 0 {
			pollInterval = time.Duration(seconds * float64(time.Second))
		}
	}
	typingInterval, err := parseDurationSecondsEnv("TELEGRAM_TYPING_INTERVAL", 4*time.Second)
	if err != nil {
		return Config{}, err
	}

	contextSize, err := parseIntEnv("TELEGRAM_CONTEXT_SIZE", 0)
	if err != nil {
		return Config{}, err
	}

	codexCommand := strings.TrimSpace(os.Getenv("CODEX_COMMAND"))
	if codexCommand == "" {
		codexCommand = "codex"
	}

	codexArgs := []string{}
	codexArgsRaw := strings.TrimSpace(os.Getenv("CODEX_ARGS"))
	if codexArgsRaw != "" {
		parsed, err := SplitArgs(codexArgsRaw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid CODEX_ARGS: %w", err)
		}
		codexArgs = parsed
	} else {
		// Default to non-interactive exec mode to avoid TTY requirements.
		codexArgs = []string{"exec", "{prompt}"}
	}

	codexPromptMode := strings.ToLower(strings.TrimSpace(os.Getenv("CODEX_PROMPT_MODE")))
	if codexPromptMode == "" {
		codexPromptMode = "arg"
	}
	if codexPromptMode != "stdin" && codexPromptMode != "arg" {
		return Config{}, fmt.Errorf("CODEX_PROMPT_MODE must be stdin or arg")
	}

	codexTimeout := 120 * time.Second
	codexTimeoutRaw := strings.TrimSpace(os.Getenv("CODEX_TIMEOUT"))
	if codexTimeoutRaw != "" {
		seconds, err := strconv.ParseFloat(codexTimeoutRaw, 64)
		if err == nil && seconds > 0 {
			codexTimeout = time.Duration(seconds * float64(time.Second))
		}
	}

	codexWorkdir := strings.TrimSpace(os.Getenv("CODEX_WORKDIR"))
	if codexWorkdir == "" {
		codexWorkdir = "."
	}

	codexDisableCPR := parseBoolEnv("CODEX_DISABLE_CPR", true)
	codexUseTTY := parseBoolEnv("CODEX_USE_TTY", false)
	codexProgressInterval, err := parseDurationSecondsEnv("CODEX_PROGRESS_INTERVAL", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	logLevel := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	if logLevel == "" {
		logLevel = "info"
	}
	if logLevel != "debug" && logLevel != "info" && logLevel != "warn" && logLevel != "error" {
		return Config{}, fmt.Errorf("LOG_LEVEL must be debug|info|warn|error")
	}

	logFile := strings.TrimSpace(os.Getenv("LOG_FILE"))
	logConsole := parseBoolEnv("LOG_CONSOLE", true)
	logColor := parseBoolEnv("LOG_COLOR", true)
	logTimeFormat := strings.TrimSpace(os.Getenv("LOG_TIME_FORMAT"))
	if logTimeFormat == "" {
		logTimeFormat = "2006-01-02 15:04:05"
	}

	return Config{
		TelegramBotToken:       token,
		TelegramAllowedChatID:  allowedChat,
		TelegramPollInterval:   pollInterval,
		TelegramTypingInterval: typingInterval,
		TelegramContextSize:    contextSize,
		CodexCommand:           codexCommand,
		CodexArgs:              codexArgs,
		CodexPromptMode:        codexPromptMode,
		CodexTimeout:           codexTimeout,
		CodexWorkdir:           codexWorkdir,
		CodexDisableCPR:        codexDisableCPR,
		CodexUseTTY:            codexUseTTY,
		CodexProgressInterval:  codexProgressInterval,
		LogLevel:               logLevel,
		LogFile:                logFile,
		LogConsole:             logConsole,
		LogColor:               logColor,
		LogTimeFormat:          logTimeFormat,
	}, nil
}

func parseBoolEnv(key string, defaultValue bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	value = strings.ToLower(value)
	switch value {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return defaultValue
	}
}

func parseDurationSecondsEnv(key string, defaultValue time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue, nil
	}
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number (seconds)", key)
	}
	if seconds <= 0 {
		return 0, nil
	}
	return time.Duration(seconds * float64(time.Second)), nil
}

func parseIntEnv(key string, defaultValue int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	if parsed < 0 {
		return 0, fmt.Errorf("%s must be >= 0", key)
	}
	return parsed, nil
}
