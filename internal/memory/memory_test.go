package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAddEntryCreatesAndAppends(t *testing.T) {
	root := t.TempDir()
	manager := NewManager(root)
	fixed := time.Date(2026, 2, 3, 12, 34, 0, 0, time.UTC)
	manager.Now = func() time.Time { return fixed }

	path, err := manager.AddEntry("hello world")
	if err != nil {
		t.Fatalf("add entry: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "## Context") {
		t.Fatalf("context header missing")
	}
	if !strings.Contains(content, "- [12:34] hello world") {
		t.Fatalf("entry missing: %s", content)
	}
}

func TestSearchMatches(t *testing.T) {
	root := t.TempDir()
	manager := NewManager(root)
	manager.Now = func() time.Time { return time.Date(2026, 2, 3, 9, 0, 0, 0, time.UTC) }

	if err := os.MkdirAll(filepath.Join(root, "memory"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	file1 := filepath.Join(root, "memory", "2026-02-02.md")
	file2 := filepath.Join(root, "memory", "2026-02-03.md")
	if err := os.WriteFile(file1, []byte("Line one\nAlpha keyword\n"), 0o644); err != nil {
		t.Fatalf("write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("Another KEYWORD here\nNo match\n"), 0o644); err != nil {
		t.Fatalf("write file2: %v", err)
	}

	matches, err := manager.Search("keyword", 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	if matches[0].File != "2026-02-02.md" || matches[1].File != "2026-02-03.md" {
		t.Fatalf("unexpected order: %+v", matches)
	}
}

func TestTodaySummaryLines(t *testing.T) {
	root := t.TempDir()
	manager := NewManager(root)
	manager.Now = func() time.Time { return time.Date(2026, 2, 3, 8, 0, 0, 0, time.UTC) }

	if err := os.MkdirAll(filepath.Join(root, "memory"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(root, "memory", "2026-02-03.md")
	content := "# 2026-02-03\n\n## Summary\n- First\n- Second\n\n## Decisions\n- Decision\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	lines, err := manager.TodaySummaryLines(20)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "- First" || lines[1] != "- Second" {
		t.Fatalf("unexpected lines: %+v", lines)
	}
}
