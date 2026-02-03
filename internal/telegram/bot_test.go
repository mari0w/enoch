package telegram

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type chatActionPayload struct {
	ChatID int64  `json:"chat_id"`
	Action string `json:"action"`
}

func TestSendChatAction(t *testing.T) {
	var got chatActionPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sendChatAction" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(body, &got); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	bot := &Bot{
		client:  server.Client(),
		baseURL: server.URL,
	}

	if err := bot.sendChatAction(42, "typing"); err != nil {
		t.Fatalf("sendChatAction error: %v", err)
	}

	if got.ChatID != 42 {
		t.Fatalf("expected chat_id 42, got %d", got.ChatID)
	}
	if got.Action != "typing" {
		t.Fatalf("expected action typing, got %q", got.Action)
	}
}
