// Package config loads worker configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	SupabaseURL        string
	SupabaseServiceKey string

	AnthropicAPIKey string
	AnthropicModel  string

	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string
	FeatureSMSAlerts bool
	FeatureLiveFeed  bool // false = mock feed source, true = real RSS feeds
	FeedURLs         []string
	PollInterval     time.Duration
	AnalysisWorkers  int
	HealthServerPort string
}

func Load() (*Config, error) {
	cfg := &Config{
		SupabaseURL:        os.Getenv("SUPABASE_URL"),
		SupabaseServiceKey: os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		AnthropicAPIKey:    os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicModel:     getenvDefault("ANTHROPIC_MODEL", "claude-haiku-4-5-20251001"),
		TwilioAccountSID:   os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:    os.Getenv("TWILIO_AUTH_TOKEN"),
		TwilioFromNumber:   os.Getenv("TWILIO_FROM_NUMBER"),
		FeatureSMSAlerts:   getenvBool("FEATURE_SMS_ALERTS", false),
		FeatureLiveFeed:    getenvBool("FEATURE_LIVE_FEED", false),
		FeedURLs:           splitCSV(getenvDefault("FEED_URLS", "")),
		HealthServerPort:   getenvDefault("PORT", "8090"),
	}

	pollSeconds, err := strconv.Atoi(getenvDefault("FEED_POLL_INTERVAL_SECONDS", "30"))
	if err != nil {
		return nil, fmt.Errorf("invalid FEED_POLL_INTERVAL_SECONDS: %w", err)
	}
	cfg.PollInterval = time.Duration(pollSeconds) * time.Second

	workers, err := strconv.Atoi(getenvDefault("ANALYSIS_WORKERS", "4"))
	if err != nil {
		return nil, fmt.Errorf("invalid ANALYSIS_WORKERS: %w", err)
	}
	cfg.AnalysisWorkers = workers

	if cfg.SupabaseURL == "" || cfg.SupabaseServiceKey == "" {
		return nil, fmt.Errorf("SUPABASE_URL and SUPABASE_SERVICE_ROLE_KEY are required")
	}
	if cfg.AnthropicAPIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is required")
	}
	if cfg.FeatureSMSAlerts && (cfg.TwilioAccountSID == "" || cfg.TwilioAuthToken == "" || cfg.TwilioFromNumber == "") {
		return nil, fmt.Errorf("FEATURE_SMS_ALERTS=true requires TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, and TWILIO_FROM_NUMBER")
	}
	if cfg.FeatureLiveFeed && len(cfg.FeedURLs) == 0 {
		return nil, fmt.Errorf("FEATURE_LIVE_FEED=true requires at least one URL in FEED_URLS")
	}

	return cfg, nil
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func splitCSV(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
