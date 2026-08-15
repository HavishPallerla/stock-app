// Package twilio is a minimal client for the Twilio Programmable Messaging
// REST API — just enough to send a single SMS, so we don't need the full SDK.
package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	accountSID string
	authToken  string
	fromNumber string
	http       *http.Client
}

func NewClient(accountSID, authToken, fromNumber string) *Client {
	return &Client{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		http:       &http.Client{Timeout: 8 * time.Second},
	}
}

type errorResponse struct {
	Message string `json:"message"`
}

// SendSMS sends a message to `to` and returns the Twilio message SID on
// success.
func (c *Client) SendSMS(ctx context.Context, to, body string) (string, error) {
	endpoint := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", c.accountSID)

	form := url.Values{
		"To":   {to},
		"From": {c.fromNumber},
		"Body": {body},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.accountSID, c.authToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		var apiErr errorResponse
		_ = json.Unmarshal(respBytes, &apiErr)
		if apiErr.Message != "" {
			return "", fmt.Errorf("twilio error (status %d): %s", resp.StatusCode, apiErr.Message)
		}
		return "", fmt.Errorf("twilio error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var ok struct {
		SID string `json:"sid"`
	}
	if err := json.Unmarshal(respBytes, &ok); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return ok.SID, nil
}
