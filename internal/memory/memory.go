package memory

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultTemplate = "# {{date}}\n\n## Summary\n- \n\n## Decisions\n- \n\n## TODOs\n- \n\n## Context\n- \n\n## Prompts/Rules\n- \n"

var errMissingSummary = errors.New("summary section not found")

// Manager provides helpers for reading/writing memory files.
type Manager struct {
	Root         string
	MemoryDir    string
	TemplatePath string
	Now          func() time.Time
}

type Match struct {
	File string
	Line int
	Text string
}

func NewManager(root string) *Manager {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	return &Manager{
		Root:         root,
		MemoryDir:    filepath.Join(root, "memory"),
		TemplatePath: filepath.Join(root, "skills", "memory", "MEMORY_TEMPLATE.md"),
		Now:          time.Now,
	}
}

func (m *Manager) TodayDate() string {
	return m.Now().Format("2006-01-02")
}

func (m *Manager) TodayFilePath() string {
	return filepath.Join(m.MemoryDir, m.TodayDate()+".md")
}

func (m *Manager) EnsureTodayFile() (string, error) {
	if err := os.MkdirAll(m.MemoryDir, 0o755); err != nil {
		return "", err
	}
	path := m.TodayFilePath()
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	template, err := m.loadTemplate()
	if err != nil {
		return "", err
	}
	content := strings.ReplaceAll(template, "{{date}}", m.TodayDate())
	content = strings.TrimRight(content, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (m *Manager) AddEntry(message string) (string, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return "", fmt.Errorf("message is empty")
	}
	path, err := m.EnsureTodayFile()
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	timestamp := m.Now().Format("15:04")
	entry := fmt.Sprintf("- [%s] %s", timestamp, message)
	updated := insertIntoContext(string(content), entry)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (m *Manager) Search(keyword string, limit int) ([]Match, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("keyword is empty")
	}
	entries, err := filepath.Glob(filepath.Join(m.MemoryDir, "*.md"))
	if err != nil {
		return nil, err
	}
	sort.Strings(entries)
	needle := strings.ToLower(keyword)
	matches := make([]Match, 0, limit)

	for _, path := range entries {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(string(content), "\n")
		for idx, line := range lines {
			if strings.Contains(strings.ToLower(line), needle) {
				matches = append(matches, Match{
					File: filepath.Base(path),
					Line: idx + 1,
					Text: truncateLine(strings.TrimSpace(line), 200),
				})
				if limit > 0 && len(matches) >= limit {
					return matches, nil
				}
			}
		}
	}

	return matches, nil
}

func (m *Manager) TodaySummaryLines(maxLines int) ([]string, error) {
	path := m.TodayFilePath()
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	var summary []string
	inSummary := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			if inSummary {
				break
			}
			if trimmed == "## Summary" {
				inSummary = true
			}
			continue
		}
		if !inSummary {
			continue
		}
		if trimmed == "" {
			continue
		}
		summary = append(summary, trimmed)
		if maxLines > 0 && len(summary) >= maxLines {
			break
		}
	}
	if !inSummary {
		return nil, errMissingSummary
	}
	return summary, nil
}

func (m *Manager) loadTemplate() (string, error) {
	data, err := os.ReadFile(m.TemplatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultTemplate, nil
		}
		return "", err
	}
	template := parseTemplate(string(data))
	if strings.TrimSpace(template) == "" {
		return defaultTemplate, nil
	}
	return template, nil
}

func parseTemplate(text string) string {
	startMarker := "<!-- TEMPLATE START -->"
	endMarker := "<!-- TEMPLATE END -->"
	start := strings.Index(text, startMarker)
	if start < 0 {
		return defaultTemplate
	}
	start += len(startMarker)
	end := strings.Index(text[start:], endMarker)
	if end < 0 {
		return defaultTemplate
	}
	template := text[start : start+end]
	return strings.Trim(template, "\n")
}

func insertIntoContext(text, entry string) string {
	lines := strings.Split(text, "\n")
	header := "## Context"
	for i, line := range lines {
		if strings.TrimSpace(line) == header {
			insertAt := i + 1
			for insertAt < len(lines) && strings.TrimSpace(lines[insertAt]) == "" {
				insertAt++
			}
			lines = append(lines[:insertAt], append([]string{entry}, lines[insertAt:]...)...)
			return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
		}
	}
	return strings.TrimRight(text, "\n") + "\n\n" + entry + "\n"
}

func truncateLine(text string, limit int) string {
	if limit <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}
