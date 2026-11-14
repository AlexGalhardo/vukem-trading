package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const telegramAPI = "https://api.telegram.org"

// SendAlert intentionally never returns the URL/token or response body in errors.
func SendAlert(ctx context.Context, token, chat, message string) error {
	return SendMessage(ctx, telegramAPI, token, chat, "vukem-trading PAPER: "+message)
}

// SendMessage posts text to a Telegram chat; with no token or chat it does nothing. apiBase is the Telegram
// Bot API origin and only api.telegram.org or a loopback address (tests) is accepted. Errors never carry the
// URL, the token or the response body.
func SendMessage(ctx context.Context, apiBase, token, chat, text string) error {
	if token == "" || chat == "" {
		return nil
	}
	if strings.ContainsAny(token, "/\r\n ?#") {
		return errors.New("invalid Telegram token format")
	}
	u, err := url.Parse(apiBase)
	if err != nil || u.Host == "" || (u.Hostname() != "api.telegram.org" && !isLoopback(u.Hostname())) {
		return errors.New("invalid Telegram API address")
	}
	payload, _ := json.Marshal(map[string]string{"chat_id": chat, "text": text})
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		u.Scheme+"://"+u.Host+"/bot"+token+"/sendMessage",
		bytes.NewReader(payload),
	)
	if err != nil {
		return errors.New("alert request failed")
	}
	req.Header.Set("Content-Type", "application/json")
	client := http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("alert delivery failed")
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("alert HTTP %d", resp.StatusCode)
	}
	return nil
}
