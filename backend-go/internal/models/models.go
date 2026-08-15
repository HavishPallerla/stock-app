// Package models mirrors the Supabase schema (see /supabase/migrations).
package models

import "time"

// Tweet mirrors the public.tweets table. Despite the name, rows currently come
// from RSS feeds (see internal/feed) rather than literal tweets; SourceAccount
// holds the feed/publisher name.
type Tweet struct {
	ID            string     `json:"id,omitempty"`
	ExternalID    string     `json:"external_id"`
	SourceAccount string     `json:"source_account"`
	Title         string     `json:"title,omitempty"`
	Content       string     `json:"content"`
	URL           string     `json:"url,omitempty"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	ScrapedAt     *time.Time `json:"scraped_at,omitempty"`
	Processed     bool       `json:"processed"`
}

// TweetAnalysis mirrors the public.tweet_analysis table.
type TweetAnalysis struct {
	ID            string    `json:"id,omitempty"`
	TweetID       string    `json:"tweet_id"`
	Ticker        string    `json:"ticker"`
	Signal        string    `json:"signal"` // buy | sell | hold
	Confidence    float64   `json:"confidence"`
	Justification string    `json:"justification"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
}

// User mirrors the public.users table joined with alert preferences, as
// returned by supa.MatchingUsers for alert fan-out.
type AlertRecipient struct {
	UserID      string   `json:"id"`
	Phone       string   `json:"phone"`
	SMSEnabled  bool     `json:"sms_enabled"`
	SignalTypes []string `json:"signal_types"`
}

// SMSLog mirrors the public.sms_log table.
type SMSLog struct {
	ID              string    `json:"id,omitempty"`
	UserID          string    `json:"user_id"`
	TweetAnalysisID string    `json:"tweet_analysis_id"`
	Status          string    `json:"status"` // sent | failed | skipped
	ErrorMessage    string    `json:"error_message,omitempty"`
	SentAt          time.Time `json:"sent_at,omitempty"`
}
