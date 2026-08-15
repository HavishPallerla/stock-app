// Package claude calls the Anthropic Messages API with a forced tool call so
// responses are always structured JSON, never free text that needs parsing.
package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const apiURL = "https://api.anthropic.com/v1/messages"
const anthropicVersion = "2023-06-01"

type Client struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey: apiKey,
		model:  model,
		// The pipeline has its own end-to-end latency budget (~2.5s/tweet), so
		// this timeout is deliberately tight: a hung call should fail fast
		// rather than stall the whole worker pool.
		http: &http.Client{Timeout: 8 * time.Second},
	}
}

// Signal is one ticker's worth of trading-signal analysis.
type Signal struct {
	Ticker        string  `json:"ticker"`
	Signal        string  `json:"signal"` // buy | sell | hold
	Confidence    float64 `json:"confidence"`
	Justification string  `json:"justification"`
}

const systemPrompt = `You are a financial news analyst. Given a single news headline and snippet, ` +
	`identify every publicly traded stock ticker it concerns and, for each one, emit a trading ` +
	`signal (buy, sell, or hold) with a confidence score between 0 and 1 and a one-to-two sentence ` +
	`justification grounded only in the text provided. If the item does not concern a specific ` +
	`publicly traded ticker, return an empty signals array. Do not speculate beyond the text.`

const toolName = "emit_signals"

var toolSchema = map[string]any{
	"name":        toolName,
	"description": "Emit trading signal analysis for a financial news item.",
	"input_schema": map[string]any{
		"type": "object",
		"properties": map[string]any{
			"signals": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"ticker": map[string]any{
							"type":        "string",
							"description": "Stock ticker symbol, e.g. AAPL",
						},
						"signal": map[string]any{
							"type": "string",
							"enum": []string{"buy", "sell", "hold"},
						},
						"confidence": map[string]any{
							"type":    "number",
							"minimum": 0,
							"maximum": 1,
						},
						"justification": map[string]any{
							"type":        "string",
							"description": "1-2 sentence rationale grounded in the source text",
						},
					},
					"required": []string{"ticker", "signal", "confidence", "justification"},
				},
			},
		},
		"required": []string{"signals"},
	},
}

type messageRequest struct {
	Model      string    `json:"model"`
	MaxTokens  int       `json:"max_tokens"`
	System     string    `json:"system"`
	Messages   []message `json:"messages"`
	Tools      []any     `json:"tools"`
	ToolChoice any       `json:"tool_choice"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messageResponse struct {
	Content []contentBlock `json:"content"`
	Error   *apiError      `json:"error"`
}

type contentBlock struct {
	Type  string          `json:"type"`
	Input json.RawMessage `json:"input"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// AnalyzeItem sends a news item's title+content to Claude and returns the
// structured signals it emits. An item mentioning no tickers returns an
// empty slice, not an error.
func (c *Client) AnalyzeItem(ctx context.Context, title, content string) ([]Signal, error) {
	text := title
	if content != "" {
		text = title + "\n\n" + content
	}

	reqBody := messageRequest{
		Model:     c.model,
		MaxTokens: 512,
		System:    systemPrompt,
		Messages:  []message{{Role: "user", Content: text}},
		Tools:     []any{toolSchema},
		ToolChoice: map[string]any{
			"type": "tool",
			"name": toolName,
		},
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var parsed messageResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("anthropic api error: %s", parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic api returned status %d", resp.StatusCode)
	}

	for _, block := range parsed.Content {
		if block.Type != "tool_use" {
			continue
		}
		var payload struct {
			Signals []Signal `json:"signals"`
		}
		if err := json.Unmarshal(block.Input, &payload); err != nil {
			return nil, fmt.Errorf("decode tool input: %w", err)
		}
		return payload.Signals, nil
	}

	return nil, fmt.Errorf("no tool_use block in response")
}
