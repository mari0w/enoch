package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"enoch/internal/codex"
	"enoch/internal/config"
	"enoch/internal/logging"
	"enoch/internal/memory"
)

type Bot struct {
	config       config.Config
	codex        *codex.Client
	client       *http.Client
	baseURL      string
	logger       *logging.Logger
	queue        chan job
	paused       bool
	running      bool
	currentTrace string
	memory       *memory.Manager
	stateMu      sync.Mutex
	contextMu    sync.Mutex
	context      map[int64][]contextEntry
	workerOnce   sync.Once
}

type job struct {
	chatID int64
	text   string
	trace  string
}

type contextEntry struct {
	role string
	text string
}

func New(cfg config.Config, codexClient *codex.Client, logger *logging.Logger) *Bot {
	client := &http.Client{Timeout: 70 * time.Second}
	root, err := os.Getwd()
	if err != nil {
		root = "."
		if logger != nil {
			logger.Warnf("telegram bot getwd failed: %v", err)
		}
	}
	return &Bot{
		config:  cfg,
		codex:   codexClient,
		client:  client,
		baseURL: "https://api.telegram.org/bot" + cfg.TelegramBotToken,
		logger:  logger,
		queue:   make(chan job, 64),
		context: map[int64][]contextEntry{},
		memory:  memory.NewManager(root),
	}
}

func (b *Bot) Run() {
	b.startWorker()
	var offset *int
	backoff := b.pollInterval()
	for {
		updates, err := b.getUpdates(offset)
		if err != nil {
			if b.logger != nil {
				b.logger.Errorf("telegram getUpdates failed: %v", err)
			}
			time.Sleep(backoff)
			backoff = nextBackoff(backoff, 60*time.Second)
			continue
		}
		backoff = b.pollInterval()

		if b.logger != nil {
			b.logger.Debugf("telegram getUpdates ok: count=%d", len(updates))
		}

		for _, update := range updates {
			id := update.UpdateID + 1
			offset = &id

			trace := fmt.Sprintf("update_id=%d", update.UpdateID)

			msg := update.Message
			if msg == nil {
				msg = update.EditedMessage
			}
			if msg == nil {
				if b.logger != nil {
					b.logger.Warnf("telegram update ignored: %s reason=no_message", trace)
				}
				continue
			}
			if msg.Text == "" {
				if b.logger != nil {
					b.logger.Warnf("telegram message ignored: %s chat_id=%d reason=empty_text", trace, msg.Chat.ID)
				}
				continue
			}

			chatID := msg.Chat.ID
			if b.logger != nil {
				preview := truncateText(msg.Text, 160)
				b.logger.Infof("telegram message received: %s chat_id=%d text=%q", trace, chatID, preview)
			}

			if !isAllowedChat(b.config.TelegramAllowedChatID, chatID) {
				if b.logger != nil {
					b.logger.Warnf("telegram message ignored: %s chat_id=%d allowed=%q", trace, chatID, b.config.TelegramAllowedChatID)
				}
				continue
			}

			if b.handleCommand(chatID, msg.Text, trace) {
				continue
			}

			if b.enqueueJob(chatID, msg.Text, trace) {
				ack := "已加入队列，请稍候。"
				if b.isPaused() {
					ack = "已暂停处理，任务已排队。"
				}
				if err := b.sendMessage(chatID, ack); err != nil {
					if b.logger != nil {
						b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
					}
				}
			} else {
				if err := b.sendMessage(chatID, "队列已满，请稍后再试。"); err != nil {
					if b.logger != nil {
						b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
					}
				}
			}
		}

		time.Sleep(b.pollInterval())
	}
}

func (b *Bot) startWorker() {
	b.workerOnce.Do(func() {
		go b.workerLoop()
	})
}

func (b *Bot) workerLoop() {
	for job := range b.queue {
		b.waitForResume()
		b.processJob(job)
	}
}

func (b *Bot) waitForResume() {
	for {
		b.stateMu.Lock()
		paused := b.paused
		b.stateMu.Unlock()
		if !paused {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (b *Bot) enqueueJob(chatID int64, text, trace string) bool {
	select {
	case b.queue <- job{chatID: chatID, text: text, trace: trace}:
		return true
	default:
		return false
	}
}

func (b *Bot) processJob(job job) {
	b.setRunning(true, job.trace)
	defer b.setRunning(false, "")

	stopTyping := b.startTypingLoop(job.chatID, job.trace)
	stopProgress := b.startProgressLoop(job.chatID, job.trace)

	start := time.Now()
	if b.logger != nil {
		b.logger.Infof("codex start: %s", job.trace)
	}

	prompt := b.buildPrompt(job.chatID, job.text)
	reply, err := b.codex.Run(prompt)

	stopTyping()
	stopProgress()

	duration := time.Since(start)
	if err != nil {
		if b.logger != nil {
			b.logger.Errorf("codex failed: %s duration=%s err=%v", job.trace, duration, err)
		}
		reply = "处理失败，请稍后重试。"
	} else if b.logger != nil {
		b.logger.Infof("codex ok: %s duration=%s bytes=%d", job.trace, duration, len(reply))
	}

	if strings.TrimSpace(reply) == "" {
		if b.logger != nil {
			b.logger.Warnf("codex empty reply: %s duration=%s", job.trace, duration)
		}
		return
	}

	if err := b.sendReply(job.chatID, reply); err != nil {
		if b.logger != nil {
			b.logger.Errorf("telegram sendMessage failed: %s err=%v", job.trace, err)
		}
		return
	}

	b.appendContext(job.chatID, "User", job.text)
	b.appendContext(job.chatID, "Assistant", reply)

	if b.logger != nil {
		b.logger.Infof("telegram reply sent: %s chat_id=%d bytes=%d", job.trace, job.chatID, len(reply))
	}
}

func (b *Bot) handleCommand(chatID int64, text, trace string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || trimmed[0] != '/' {
		return false
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return false
	}
	cmd := parts[0]
	if cmd == "/memory" && len(parts) >= 2 {
		cmd = "/memory_" + strings.ToLower(parts[1])
		parts = append([]string{cmd}, parts[2:]...)
	}

	switch cmd {
	case "/status":
		status := b.statusSummary()
		if err := b.sendMessage(chatID, status); err != nil && b.logger != nil {
			b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
		}
		return true
	case "/stop":
		b.setPaused(true)
		if err := b.sendMessage(chatID, "已暂停处理新任务。"); err != nil && b.logger != nil {
			b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
		}
		return true
	case "/resume":
		b.setPaused(false)
		if err := b.sendMessage(chatID, "已恢复处理。"); err != nil && b.logger != nil {
			b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
		}
		return true
	case "/reset":
		b.resetContext(chatID)
		if err := b.sendMessage(chatID, "已清空会话上下文。"); err != nil && b.logger != nil {
			b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
		}
		return true
	case "/memory_add":
		message := strings.TrimSpace(strings.Join(parts[1:], " "))
		if message == "" {
			if err := b.sendMessage(chatID, "用法: /memory_add 记录内容 (或 /memory add 记录内容)"); err != nil && b.logger != nil {
				b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
			}
			return true
		}
		path, err := b.memory.AddEntry(message)
		if err != nil {
			if b.logger != nil {
				b.logger.Errorf("memory add failed: %s err=%v", trace, err)
			}
			if err := b.sendMessage(chatID, "写入记忆失败，请稍后重试。"); err != nil && b.logger != nil {
				b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
			}
			return true
		}
		ack := fmt.Sprintf("已写入 %s", filepathBase(path))
		if err := b.sendMessage(chatID, ack); err != nil && b.logger != nil {
			b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
		}
		return true
	case "/memory_search":
		keyword := strings.TrimSpace(strings.Join(parts[1:], " "))
		if keyword == "" {
			if err := b.sendMessage(chatID, "用法: /memory_search 关键词 (或 /memory search 关键词)"); err != nil && b.logger != nil {
				b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
			}
			return true
		}
		matches, err := b.memory.Search(keyword, 5)
		if err != nil {
			if b.logger != nil {
				b.logger.Errorf("memory search failed: %s err=%v", trace, err)
			}
			if err := b.sendMessage(chatID, "检索失败，请稍后重试。"); err != nil && b.logger != nil {
				b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
			}
			return true
		}
		if len(matches) == 0 {
			if err := b.sendMessage(chatID, "未找到匹配结果。"); err != nil && b.logger != nil {
				b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
			}
			return true
		}
		resultText := formatSearchResults(matches)
		if err := b.sendTextOrDocument(chatID, "memory_search.txt", resultText); err != nil && b.logger != nil {
			b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
		}
		return true
	case "/memory_today":
		lines, err := b.memory.TodaySummaryLines(20)
		if err != nil {
			if b.logger != nil {
				b.logger.Errorf("memory today failed: %s err=%v", trace, err)
			}
			if err := b.sendMessage(chatID, "今日记忆文件不存在或无法读取。"); err != nil && b.logger != nil {
				b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
			}
			return true
		}
		if len(lines) == 0 {
			if err := b.sendMessage(chatID, "今日记忆摘要为空。"); err != nil && b.logger != nil {
				b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
			}
			return true
		}
		result := fmt.Sprintf("今日记忆摘要 (%s):\n%s", b.memory.TodayDate(), strings.Join(lines, "\n"))
		if err := b.sendMessage(chatID, result); err != nil && b.logger != nil {
			b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
		}
		return true
	default:
		return false
	}
}

func (b *Bot) setPaused(paused bool) {
	b.stateMu.Lock()
	b.paused = paused
	b.stateMu.Unlock()
}

func (b *Bot) isPaused() bool {
	b.stateMu.Lock()
	defer b.stateMu.Unlock()
	return b.paused
}

func (b *Bot) setRunning(running bool, trace string) {
	b.stateMu.Lock()
	b.running = running
	b.currentTrace = trace
	b.stateMu.Unlock()
}

func (b *Bot) statusSummary() string {
	b.stateMu.Lock()
	paused := b.paused
	running := b.running
	trace := b.currentTrace
	queueLen := len(b.queue)
	b.stateMu.Unlock()

	contextSize := b.config.TelegramContextSize
	contextCount := b.contextCount()

	status := "状态："
	if paused {
		status += "已暂停"
	} else {
		status += "运行中"
	}
	runningText := "否"
	if running {
		runningText = "是"
	}
	if trace == "" {
		trace = "-"
	}
	return fmt.Sprintf("%s\n处理中：%s\n队列长度：%d\n当前任务：%s\n上下文大小：%d\n上下文条目：%d",
		status, runningText, queueLen, trace, contextSize, contextCount)
}

func (b *Bot) pollInterval() time.Duration {
	if b.config.TelegramPollInterval <= 0 {
		return 2 * time.Second
	}
	return b.config.TelegramPollInterval
}

func nextBackoff(current, max time.Duration) time.Duration {
	if current <= 0 {
		return max
	}
	next := current * 2
	if next > max {
		return max
	}
	return next
}

type updatesResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

type Update struct {
	UpdateID      int      `json:"update_id"`
	Message       *Message `json:"message"`
	EditedMessage *Message `json:"edited_message"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	Text      string `json:"text"`
	Chat      Chat   `json:"chat"`
}

type Chat struct {
	ID int64 `json:"id"`
}

func (b *Bot) getUpdates(offset *int) ([]Update, error) {
	payload := map[string]interface{}{
		"timeout": 30,
	}
	if offset != nil {
		payload["offset"] = *offset
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, b.baseURL+"/getUpdates", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("getUpdates status: %s", resp.Status)
	}

	var decoded updatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if !decoded.Ok {
		return nil, fmt.Errorf("telegram returned ok=false")
	}
	return decoded.Result, nil
}

func (b *Bot) sendMessage(chatID int64, text string) error {
	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, b.baseURL+"/sendMessage", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("sendMessage status: %s", resp.Status)
	}
	return nil
}

func (b *Bot) sendReply(chatID int64, text string) error {
	const limit = 4096
	if len([]rune(text)) <= limit {
		return b.sendMessage(chatID, text)
	}

	chunks := splitMessage(text, limit)
	if len(chunks) > 3 {
		return b.sendDocument(chatID, "reply.txt", []byte(text))
	}
	for _, chunk := range chunks {
		if err := b.sendMessage(chatID, chunk); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bot) sendTextOrDocument(chatID int64, filename, text string) error {
	const limit = 3500
	if len([]rune(text)) <= limit {
		return b.sendMessage(chatID, text)
	}
	return b.sendDocument(chatID, filename, []byte(text))
}

func (b *Bot) sendDocument(chatID int64, filename string, content []byte) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("chat_id", strconv.FormatInt(chatID, 10)); err != nil {
		return err
	}
	part, err := writer.CreateFormFile("document", filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, b.baseURL+"/sendDocument", &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("sendDocument status: %s", resp.Status)
	}
	return nil
}

func (b *Bot) sendChatAction(chatID int64, action string) error {
	payload := map[string]interface{}{
		"chat_id": chatID,
		"action":  action,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, b.baseURL+"/sendChatAction", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("sendChatAction status: %s", resp.Status)
	}
	return nil
}

func (b *Bot) startTypingLoop(chatID int64, trace string) func() {
	if b.config.TelegramTypingInterval <= 0 {
		return func() {}
	}

	if err := b.sendChatAction(chatID, "typing"); err != nil {
		if b.logger != nil {
			b.logger.Warnf("telegram sendChatAction failed: %s err=%v", trace, err)
		}
	} else if b.logger != nil {
		b.logger.Debugf("telegram sendChatAction ok: %s action=typing", trace)
	}

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(b.config.TelegramTypingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := b.sendChatAction(chatID, "typing"); err != nil {
					if b.logger != nil {
						b.logger.Warnf("telegram sendChatAction failed: %s err=%v", trace, err)
					}
				} else if b.logger != nil {
					b.logger.Debugf("telegram sendChatAction ok: %s action=typing", trace)
				}
			}
		}
	}()

	return func() { close(done) }
}

func (b *Bot) startProgressLoop(chatID int64, trace string) func() {
	interval := b.config.CodexProgressInterval
	if interval <= 0 {
		return func() {}
	}
	minInterval := 30 * time.Second
	if interval < minInterval {
		interval = minInterval
	}

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := b.sendMessage(chatID, "仍在处理中"); err != nil {
					if b.logger != nil {
						b.logger.Warnf("telegram progress update failed: %s err=%v", trace, err)
					}
				} else if b.logger != nil {
					b.logger.Debugf("telegram progress update sent: %s", trace)
				}
			}
		}
	}()

	return func() { close(done) }
}

func isAllowedChat(allowed string, chatID int64) bool {
	if strings.TrimSpace(allowed) == "" {
		return true
	}
	return strconv.FormatInt(chatID, 10) == allowed
}

func truncateText(text string, limit int) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.TrimSpace(text)
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}

func splitMessage(text string, limit int) []string {
	if limit <= 0 {
		return []string{text}
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return []string{text}
	}
	chunks := make([]string, 0, (len(runes)/limit)+1)
	for len(runes) > 0 {
		if len(runes) <= limit {
			chunks = append(chunks, string(runes))
			break
		}
		chunks = append(chunks, string(runes[:limit]))
		runes = runes[limit:]
	}
	return chunks
}

func formatSearchResults(matches []memory.Match) string {
	var sb strings.Builder
	sb.WriteString("检索结果(最多 5 条):\n")
	for _, match := range matches {
		line := fmt.Sprintf("%s:%d: %s", match.File, match.Line, match.Text)
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func filepathBase(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func (b *Bot) buildPrompt(chatID int64, text string) string {
	if b.config.TelegramContextSize <= 0 {
		return text
	}
	entries := b.getContext(chatID)
	if len(entries) == 0 {
		return text
	}
	var sb strings.Builder
	sb.WriteString("Conversation history:\n")
	for _, entry := range entries {
		sb.WriteString(entry.role)
		sb.WriteString(": ")
		sb.WriteString(entry.text)
		sb.WriteString("\n")
	}
	sb.WriteString("User: ")
	sb.WriteString(text)
	return sb.String()
}

func (b *Bot) appendContext(chatID int64, role, text string) {
	if b.config.TelegramContextSize <= 0 {
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	b.contextMu.Lock()
	defer b.contextMu.Unlock()
	entries := append(b.context[chatID], contextEntry{role: role, text: text})
	limit := b.config.TelegramContextSize
	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	b.context[chatID] = entries
}

func (b *Bot) getContext(chatID int64) []contextEntry {
	b.contextMu.Lock()
	defer b.contextMu.Unlock()
	entries := b.context[chatID]
	out := make([]contextEntry, len(entries))
	copy(out, entries)
	return out
}

func (b *Bot) resetContext(chatID int64) {
	b.contextMu.Lock()
	defer b.contextMu.Unlock()
	delete(b.context, chatID)
}

func (b *Bot) contextCount() int {
	b.contextMu.Lock()
	defer b.contextMu.Unlock()
	total := 0
	for _, entries := range b.context {
		total += len(entries)
	}
	return total
}
