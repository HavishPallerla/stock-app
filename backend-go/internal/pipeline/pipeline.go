// Package pipeline turns one newly-scraped tweet into stored analysis rows
// and, where applicable, delivered alerts — instrumenting each stage so the
// end-to-end latency budget (~2.5s/tweet) can be attributed to a specific
// stage instead of guessed at.
package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"stockapp/backend-go/internal/alerts"
	"stockapp/backend-go/internal/claude"
	"stockapp/backend-go/internal/models"
	"stockapp/backend-go/internal/supa"
)

type Processor struct {
	supa       *supa.Client
	claude     *claude.Client
	dispatcher *alerts.Dispatcher
}

func NewProcessor(supaClient *supa.Client, claudeClient *claude.Client, dispatcher *alerts.Dispatcher) *Processor {
	return &Processor{supa: supaClient, claude: claudeClient, dispatcher: dispatcher}
}

// StageTimings breaks down where time went processing a single tweet, so
// bottlenecks (network fetch vs LLM call vs DB writes vs SMS) can be
// identified from logs rather than guessed at.
type StageTimings struct {
	QueueWait time.Duration // time between scrape and this worker picking it up
	Claude    time.Duration
	DBWrite   time.Duration
	Alerts    time.Duration
	Total     time.Duration
}

// Process analyzes one tweet with Claude, persists the resulting signals,
// marks the tweet processed, and fans out alerts for each signal generated.
func (p *Processor) Process(ctx context.Context, tweet models.Tweet, enqueuedAt time.Time) (StageTimings, error) {
	start := time.Now()
	timings := StageTimings{QueueWait: start.Sub(enqueuedAt)}

	claudeStart := time.Now()
	signals, err := p.claude.AnalyzeItem(ctx, tweet.Title, tweet.Content)
	timings.Claude = time.Since(claudeStart)
	if err != nil {
		return timings, fmt.Errorf("claude analysis: %w", err)
	}

	dbStart := time.Now()
	var inserted []models.TweetAnalysis
	if len(signals) > 0 {
		rows := make([]models.TweetAnalysis, len(signals))
		for i, s := range signals {
			rows[i] = models.TweetAnalysis{
				TweetID:       tweet.ID,
				Ticker:        s.Ticker,
				Signal:        s.Signal,
				Confidence:    s.Confidence,
				Justification: s.Justification,
			}
		}
		inserted, err = p.supa.InsertAnalyses(ctx, rows)
		if err != nil {
			return timings, fmt.Errorf("insert analyses: %w", err)
		}
	}
	if err := p.supa.MarkTweetProcessed(ctx, tweet.ID); err != nil {
		return timings, fmt.Errorf("mark processed: %w", err)
	}
	timings.DBWrite = time.Since(dbStart)

	alertStart := time.Now()
	for _, analysis := range inserted {
		if err := p.dispatcher.Dispatch(ctx, analysis); err != nil {
			slog.Warn("pipeline: alert dispatch failed", "tweet_id", tweet.ID, "ticker", analysis.Ticker, "error", err)
		}
	}
	timings.Alerts = time.Since(alertStart)

	timings.Total = time.Since(start)
	slog.Info("pipeline: processed tweet",
		"tweet_id", tweet.ID,
		"signals", len(signals),
		"queue_wait_ms", timings.QueueWait.Milliseconds(),
		"claude_ms", timings.Claude.Milliseconds(),
		"db_write_ms", timings.DBWrite.Milliseconds(),
		"alerts_ms", timings.Alerts.Milliseconds(),
		"total_ms", timings.Total.Milliseconds(),
	)

	return timings, nil
}
