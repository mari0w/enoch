package telegram

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type chatActionPayload struct {
	ChatID int64  `json:"chat_id"`
	Action string `json:"action"`
}

func TestSendChatAction(t *testing.T) {
	var got chatActionPayload
	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/sendChatAction" {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewBufferString("")),
					Header:     make(http.Header),
				}, nil
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewBufferString("")),
					Header:     make(http.Header),
				}, nil
			}
			if err := json.Unmarshal(body, &got); err != nil {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewBufferString("")),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	bot := &Bot{
		client:  client,
		baseURL: "http://example.com",
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
