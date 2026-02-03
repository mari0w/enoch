package codex

import (
	"errors"
	"strings"
	"testing"
)

func TestReplacePromptPlaceholder(t *testing.T) {
	args := []string{"--prompt", "{prompt}", "--flag"}
	out, used := replacePromptPlaceholder(args, "hello")
	if !used {
		t.Fatalf("expected placeholder to be used")
	}
	if len(out) != 3 {
		t.Fatalf("len mismatch: %d", len(out))
	}
	if out[1] != "hello" {
		t.Fatalf("expected prompt replaced, got %q", out[1])
	}
}

func TestShellJoinQuotes(t *testing.T) {
	args := []string{"codex", "--prompt", "hello world", "it's"}
	joined := shellJoin(args)
	if joined == "" {
		t.Fatalf("expected non-empty join")
	}
	if joined == "codex --prompt hello world it's" {
		t.Fatalf("expected quoting, got %q", joined)
	}
}

func TestBuildCommandErrorPrefersStderr(t *testing.T) {
	err := buildCommandError("stdout", "stderr", errors.New("boom"))
	if !strings.Contains(err.Error(), "stderr") {
		t.Fatalf("expected stderr in error, got %q", err.Error())
	}
}

func TestBuildCommandErrorUsesStdoutWhenNoStderr(t *testing.T) {
	err := buildCommandError("stdout", "", errors.New("boom"))
	if !strings.Contains(err.Error(), "stdout") {
		t.Fatalf("expected stdout in error, got %q", err.Error())
	}
}

func TestBuildCommandErrorFallsBackToErr(t *testing.T) {
	err := buildCommandError("", "", errors.New("boom"))
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected base error in message, got %q", err.Error())
	}
}

func TestIsTTYError(t *testing.T) {
	if !isTTYError(errors.New("stdin is not a terminal")) {
		t.Fatalf("expected stdin error to be detected")
	}
	if !isTTYError(errors.New("The cursor position could not be read within a normal duration")) {
		t.Fatalf("expected cursor position error to be detected")
	}
	if isTTYError(errors.New("other error")) {
		t.Fatalf("unexpected tty error detection")
	}
}

func TestTruncatePrompt(t *testing.T) {
	if got := truncatePrompt("hello", 10); got != "hello" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := truncatePrompt("hello", 3); got != "hel..." {
		t.Fatalf("unexpected: %q", got)
	}
	if got := truncatePrompt("\n  hi  \n", 10); got != "hi" {
		t.Fatalf("unexpected: %q", got)
	}
}
