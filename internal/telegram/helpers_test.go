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

func TestSplitMessage(t *testing.T) {
	chunks := splitMessage("hello", 10)
	if len(chunks) != 1 || chunks[0] != "hello" {
		t.Fatalf("unexpected chunks: %#v", chunks)
	}

	chunks = splitMessage("abcdef", 2)
	if len(chunks) != 3 {
		t.Fatalf("unexpected chunk count: %d", len(chunks))
	}
	if chunks[0] != "ab" || chunks[1] != "cd" || chunks[2] != "ef" {
		t.Fatalf("unexpected chunks: %#v", chunks)
	}
}
