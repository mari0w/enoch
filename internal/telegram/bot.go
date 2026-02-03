package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"enoch/internal/codex"
	"enoch/internal/config"
	"enoch/internal/logging"
)

type Bot struct {
	config  config.Config
	codex   *codex.Client
	client  *http.Client
	baseURL string
	logger  *logging.Logger
}

func New(cfg config.Config, codexClient *codex.Client, logger *logging.Logger) *Bot {
	client := &http.Client{Timeout: 70 * time.Second}
	return &Bot{
		config:  cfg,
		codex:   codexClient,
		client:  client,
		baseURL: "https://api.telegram.org/bot" + cfg.TelegramBotToken,
		logger:  logger,
	}
}

func (b *Bot) Run() {
	var offset *int
	for {
		updates, err := b.getUpdates(offset)
		if err != nil {
			if b.logger != nil {
				b.logger.Errorf("telegram getUpdates failed: %v", err)
			}
			time.Sleep(b.config.TelegramPollInterval)
			continue
		}

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

			stopTyping := b.startTypingLoop(chatID, trace)

			start := time.Now()
			if b.logger != nil {
				b.logger.Infof("codex start: %s", trace)
			}
			reply, err := b.codex.Run(msg.Text)
			stopTyping()
			duration := time.Since(start)
			if err != nil {
				if b.logger != nil {
					b.logger.Errorf("codex failed: %s duration=%s err=%v", trace, duration, err)
				}
				reply = "Error: " + err.Error()
			} else if b.logger != nil {
				b.logger.Infof("codex ok: %s duration=%s bytes=%d", trace, duration, len(reply))
			}
			if strings.TrimSpace(reply) == "" {
				if b.logger != nil {
					b.logger.Warnf("codex empty reply: %s duration=%s", trace, duration)
				}
				continue
			}
			if err := b.sendMessage(chatID, reply); err != nil {
				if b.logger != nil {
					b.logger.Errorf("telegram sendMessage failed: %s err=%v", trace, err)
				}
				continue
			}
			if b.logger != nil {
				b.logger.Infof("telegram reply sent: %s chat_id=%d bytes=%d", trace, chatID, len(reply))
			}
		}

		time.Sleep(b.config.TelegramPollInterval)
	}
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
