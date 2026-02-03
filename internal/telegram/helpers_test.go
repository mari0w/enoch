package telegram

import "testing"

func TestIsAllowedChat(t *testing.T) {
	if !isAllowedChat("", 123) {
		t.Fatalf("empty allowlist should allow")
	}
	if !isAllowedChat("123", 123) {
		t.Fatalf("matching chat should allow")
	}
	if isAllowedChat("999", 123) {
		t.Fatalf("mismatched chat should not allow")
	}
}

func TestTruncateText(t *testing.T) {
	if got := truncateText("hello", 10); got != "hello" {
		t.Fatalf("unexpected: %q", got)
	}
	if got := truncateText("hello", 3); got != "hel..." {
		t.Fatalf("unexpected: %q", got)
	}
	if got := truncateText("\n  hi  \n", 10); got != "hi" {
		t.Fatalf("unexpected: %q", got)
	}
}
