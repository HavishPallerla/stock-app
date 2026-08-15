// Package supa is a minimal PostgREST client for the Supabase tables this
// worker touches. It intentionally avoids a full Supabase SDK dependency —
// every call the worker makes is a simple REST request signed with the
// service role key, which is all we need server-side.
package supa

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"stockapp/backend-go/internal/models"
)

type Client struct {
	baseURL    string
	serviceKey string
	http       *http.Client
}

func NewClient(baseURL, serviceKey string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		serviceKey: serviceKey,
		http:       &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, prefer string, body any) ([]byte, int, error) {
	u := fmt.Sprintf("%s/rest/v1/%s", c.baseURL, path)
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal body: %w", err)
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("apikey", c.serviceKey)
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("Content-Type", "application/json")
	if prefer != "" {
		req.Header.Set("Prefer", prefer)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return respBody, resp.StatusCode, fmt.Errorf("supabase %s %s -> %d: %s", method, path, resp.StatusCode, string(respBody))
	}
	return respBody, resp.StatusCode, nil
}

// InsertNewTweets bulk-inserts items, skipping any whose external_id already
// exists (unique constraint + ignore-duplicates), and returns only the rows
// that were actually newly inserted.
func (c *Client) InsertNewTweets(ctx context.Context, rows []models.Tweet) ([]models.Tweet, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	q := url.Values{"on_conflict": {"external_id"}}
	body, _, err := c.do(ctx, http.MethodPost, "tweets", q, "resolution=ignore-duplicates,return=representation", rows)
	if err != nil {
		return nil, err
	}
	var inserted []models.Tweet
	if err := json.Unmarshal(body, &inserted); err != nil {
		return nil, fmt.Errorf("decode inserted tweets: %w", err)
	}
	return inserted, nil
}

func (c *Client) MarkTweetProcessed(ctx context.Context, tweetID string) error {
	q := url.Values{"id": {"eq." + tweetID}}
	_, _, err := c.do(ctx, http.MethodPatch, "tweets", q, "return=minimal", map[string]any{"processed": true})
	return err
}

// InsertAnalyses writes one or more tweet_analysis rows (one per ticker
// mentioned in the source tweet) and returns them with server-assigned ids.
func (c *Client) InsertAnalyses(ctx context.Context, rows []models.TweetAnalysis) ([]models.TweetAnalysis, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	body, _, err := c.do(ctx, http.MethodPost, "tweet_analysis", nil, "return=representation", rows)
	if err != nil {
		return nil, err
	}
	var inserted []models.TweetAnalysis
	if err := json.Unmarshal(body, &inserted); err != nil {
		return nil, fmt.Errorf("decode inserted analyses: %w", err)
	}
	return inserted, nil
}

// MatchingRecipients returns users who are opted in for SMS alerts on this
// ticker and signal type: on the ticker's watchlist, sms_enabled, and the
// signal type is in their subscribed signal_types.
func (c *Client) MatchingRecipients(ctx context.Context, ticker, signal string) ([]models.AlertRecipient, error) {
	watchQ := url.Values{
		"ticker": {"eq." + ticker},
		"select": {"user_id"},
	}
	watchBody, _, err := c.do(ctx, http.MethodGet, "watchlist", watchQ, "", nil)
	if err != nil {
		return nil, fmt.Errorf("lookup watchlist: %w", err)
	}
	var watchers []struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(watchBody, &watchers); err != nil {
		return nil, fmt.Errorf("decode watchlist: %w", err)
	}
	if len(watchers) == 0 {
		return nil, nil
	}

	ids := make([]string, len(watchers))
	for i, w := range watchers {
		ids[i] = w.UserID
	}

	prefQ := url.Values{
		"user_id":      {"in.(" + strings.Join(ids, ",") + ")"},
		"sms_enabled":  {"eq.true"},
		"signal_types": {"cs.{" + signal + "}"},
		"select":       {"user_id,sms_enabled,signal_types,users(phone)"},
	}
	prefBody, _, err := c.do(ctx, http.MethodGet, "alert_preferences", prefQ, "", nil)
	if err != nil {
		return nil, fmt.Errorf("lookup alert_preferences: %w", err)
	}

	var rows []struct {
		UserID      string   `json:"user_id"`
		SMSEnabled  bool     `json:"sms_enabled"`
		SignalTypes []string `json:"signal_types"`
		Users       struct {
			Phone string `json:"phone"`
		} `json:"users"`
	}
	if err := json.Unmarshal(prefBody, &rows); err != nil {
		return nil, fmt.Errorf("decode alert_preferences: %w", err)
	}

	out := make([]models.AlertRecipient, 0, len(rows))
	for _, r := range rows {
		if r.Users.Phone == "" {
			continue
		}
		out = append(out, models.AlertRecipient{
			UserID:      r.UserID,
			Phone:       r.Users.Phone,
			SMSEnabled:  r.SMSEnabled,
			SignalTypes: r.SignalTypes,
		})
	}
	return out, nil
}

func (c *Client) InsertSMSLog(ctx context.Context, row models.SMSLog) error {
	_, _, err := c.do(ctx, http.MethodPost, "sms_log", nil, "return=minimal", row)
	return err
}
