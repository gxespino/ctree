package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	pollInterval = 3 * time.Second
	slackBaseURL = "https://slack.com/api"
)

// SendMessage posts a message to the configured Slack channel.
// Returns the message timestamp (thread ID) for threading replies.
func SendMessage(cfg *Config, text string) (string, error) {
	payload := map[string]any{
		"channel":      cfg.ChannelID,
		"text":         text,
		"unfurl_links": false,
		"unfurl_media": false,
	}

	resp, err := postJSON(cfg.BotToken, slackBaseURL+"/chat.postMessage", payload)
	if err != nil {
		return "", fmt.Errorf("chat.postMessage: %w", err)
	}
	if ok, _ := resp["ok"].(bool); !ok {
		errMsg, _ := resp["error"].(string)
		return "", fmt.Errorf("chat.postMessage: %s", errMsg)
	}

	ts, _ := resp["ts"].(string)
	return ts, nil
}

// ReplyInThread posts a reply in an existing thread.
func ReplyInThread(cfg *Config, threadTS, text string) error {
	payload := map[string]any{
		"channel":   cfg.ChannelID,
		"text":      text,
		"thread_ts": threadTS,
	}
	resp, err := postJSON(cfg.BotToken, slackBaseURL+"/chat.postMessage", payload)
	if err != nil {
		return fmt.Errorf("chat.postMessage: %w", err)
	}
	if ok, _ := resp["ok"].(bool); !ok {
		errMsg, _ := resp["error"].(string)
		return fmt.Errorf("chat.postMessage: %s", errMsg)
	}
	return nil
}

// WaitForReply polls for a threaded reply from a human user.
// Returns the reply text, or "" if timeout is reached (not an error).
func WaitForReply(cfg *Config, threadTS string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		reply, err := checkForReply(cfg, threadTS)
		if err != nil {
			return "", err
		}
		if reply != "" {
			return reply, nil
		}
		time.Sleep(pollInterval)
	}

	return "", nil
}

// checkForReply fetches thread replies and returns the first human reply.
func checkForReply(cfg *Config, threadTS string) (string, error) {
	params := url.Values{
		"channel": {cfg.ChannelID},
		"ts":      {threadTS},
		"limit":   {"10"},
	}

	reqURL := slackBaseURL + "/conversations.replies?" + params.Encode()
	resp, err := getJSON(cfg.BotToken, reqURL)
	if err != nil {
		return "", fmt.Errorf("conversations.replies: %w", err)
	}
	if ok, _ := resp["ok"].(bool); !ok {
		errMsg, _ := resp["error"].(string)
		return "", fmt.Errorf("conversations.replies: %s", errMsg)
	}

	messages, _ := resp["messages"].([]any)
	// messages[0] is the parent; replies are messages[1:]
	for _, m := range messages[1:] {
		msg, _ := m.(map[string]any)
		if msg == nil {
			continue
		}
		// Skip bot messages
		if _, isBot := msg["bot_id"]; isBot {
			continue
		}
		text, _ := msg["text"].(string)
		if text != "" {
			return text, nil
		}
	}

	return "", nil
}

func postJSON(token, url string, payload any) (map[string]any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return decodeJSON(resp.Body)
}

func getJSON(token, url string) (map[string]any, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return decodeJSON(resp.Body)
}

func decodeJSON(r io.Reader) (map[string]any, error) {
	var result map[string]any
	if err := json.NewDecoder(r).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}
