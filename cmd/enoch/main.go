package main

import (
	"fmt"
	"os"
	"time"

	"enoch/internal/codex"
	"enoch/internal/config"
	"enoch/internal/logging"
	"enoch/internal/telegram"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fallbackLog("config error: %v", err)
		os.Exit(1)
	}

	logger, err := logging.New(cfg)
	if err != nil {
		fallbackLog("logger init error: %v", err)
		os.Exit(1)
	}
	defer func() {
		_ = logger.Close()
	}()

	codexClient := codex.New(cfg, logger)
	bot := telegram.New(cfg, codexClient, logger)

	logger.Infof("[enoch] Telegram polling started")
	bot.Run()
}

func fallbackLog(format string, args ...interface{}) {
	ts := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	fmt.Printf("%s [ERROR] %s\n", ts, message)
}
