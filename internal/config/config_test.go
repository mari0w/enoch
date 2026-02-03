package config

import (
	"os"
	"testing"
)

func TestLoadConfigParsesArgs(t *testing.T) {
	resetEnv := setTestEnv(map[string]string{
		"TELEGRAM_BOT_TOKEN": "token",
		"CODEX_ARGS":         "--prompt {prompt} --flag",
		"CODEX_PROMPT_MODE":  "arg",
		"CODEX_DISABLE_CPR":  "false",
	})
	defer resetEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.CodexCommand != "codex" {
		t.Fatalf("CodexCommand mismatch: %s", cfg.CodexCommand)
	}
	if len(cfg.CodexArgs) != 3 {
		t.Fatalf("CodexArgs len mismatch: %d", len(cfg.CodexArgs))
	}
	if cfg.CodexArgs[0] != "--prompt" || cfg.CodexArgs[1] != "{prompt}" || cfg.CodexArgs[2] != "--flag" {
		t.Fatalf("CodexArgs unexpected: %#v", cfg.CodexArgs)
	}
	if cfg.CodexDisableCPR {
		t.Fatalf("expected CodexDisableCPR to be false")
	}
}

func setTestEnv(values map[string]string) func() {
	prev := map[string]string{}
	for key := range values {
		if val, ok := os.LookupEnv(key); ok {
			prev[key] = val
		} else {
			prev[key] = ""
		}
	}

	for key, val := range values {
		_ = os.Setenv(key, val)
	}

	return func() {
		for key, val := range prev {
			if val == "" {
				_ = os.Unsetenv(key)
				continue
			}
			_ = os.Setenv(key, val)
		}
	}
}
