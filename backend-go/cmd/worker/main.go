// Command worker runs the scrape -> analyze -> alert pipeline as a single
// long-running process. The feed poller and the analysis workers communicate
// over an in-memory channel (component 1's "queue"); scrape and analysis are
// still cleanly separated packages (internal/feed vs internal/pipeline) so
// they can be split into separate deployables later without a rewrite.
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"stockapp/backend-go/internal/alerts"
	"stockapp/backend-go/internal/claude"
	"stockapp/backend-go/internal/config"
	"stockapp/backend-go/internal/feed"
	"stockapp/backend-go/internal/models"
	"stockapp/backend-go/internal/pipeline"
	"stockapp/backend-go/internal/supa"
	"stockapp/backend-go/internal/twilio"
)

type queueItem struct {
	Tweet      models.Tweet
	EnqueuedAt time.Time
}

func main() {
	_ = godotenv.Load() // no-op if .env doesn't exist; real env vars always win

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	supaClient := supa.NewClient(cfg.SupabaseURL, cfg.SupabaseServiceKey)
	claudeClient := claude.NewClient(cfg.AnthropicAPIKey, cfg.AnthropicModel)

	var twilioClient *twilio.Client
	if cfg.FeatureSMSAlerts {
		twilioClient = twilio.NewClient(cfg.TwilioAccountSID, cfg.TwilioAuthToken, cfg.TwilioFromNumber)
	}
	dispatcher := alerts.NewDispatcher(supaClient, twilioClient, cfg.FeatureSMSAlerts)
	processor := pipeline.NewProcessor(supaClient, claudeClient, dispatcher)

	var source feed.Source
	if cfg.FeatureLiveFeed {
		source = feed.NewRSSSource(cfg.FeedURLs)
		slog.Info("using live RSS feed source", "feeds", cfg.FeedURLs)
	} else {
		mock, err := feed.NewMockSource()
		if err != nil {
			slog.Error("failed to init mock feed source", "error", err)
			os.Exit(1)
		}
		source = mock
		slog.Info("using mock feed source (set FEATURE_LIVE_FEED=true for real RSS feeds)")
	}

	queueCh := make(chan queueItem, 100)

	go runHealthServer(cfg.HealthServerPort)
	go pollLoop(ctx, source, supaClient, cfg.PollInterval, queueCh)

	var wg sync.WaitGroup
	for i := 0; i < cfg.AnalysisWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(ctx, workerID, processor, queueCh)
		}(i)
	}

	slog.Info("worker started",
		"poll_interval", cfg.PollInterval.String(),
		"analysis_workers", cfg.AnalysisWorkers,
		"sms_alerts_enabled", cfg.FeatureSMSAlerts,
	)

	<-ctx.Done()
	slog.Info("shutting down, waiting for in-flight work to drain")
	close(queueCh)
	wg.Wait()
}

func pollLoop(ctx context.Context, source feed.Source, supaClient *supa.Client, interval time.Duration, queueCh chan<- queueItem) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	poll := func() {
		items, err := source.Poll(ctx)
		if err != nil {
			slog.Warn("feed poll failed", "source", source.Name(), "error", err)
			return
		}
		if len(items) == 0 {
			return
		}

		now := time.Now()
		tweets := make([]models.Tweet, len(items))
		for i, it := range items {
			publishedAt := it.PublishedAt
			tweets[i] = models.Tweet{
				ExternalID:    it.ExternalID,
				SourceAccount: it.SourceAccount,
				Title:         it.Title,
				Content:       it.Content,
				URL:           it.URL,
				PublishedAt:   &publishedAt,
			}
		}

		inserted, err := supaClient.InsertNewTweets(ctx, tweets)
		if err != nil {
			slog.Warn("failed to persist new tweets", "error", err)
			return
		}
		if len(inserted) == 0 {
			return // everything in this batch was a duplicate we've already seen
		}

		slog.Info("enqueuing new tweets", "count", len(inserted))
		for _, t := range inserted {
			select {
			case queueCh <- queueItem{Tweet: t, EnqueuedAt: now}:
			case <-ctx.Done():
				return
			}
		}
	}

	poll() // don't wait a full interval before the first poll
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			poll()
		}
	}
}

func runWorker(ctx context.Context, workerID int, processor *pipeline.Processor, queueCh <-chan queueItem) {
	for item := range queueCh {
		if _, err := processor.Process(ctx, item.Tweet, item.EnqueuedAt); err != nil {
			slog.Error("pipeline processing failed", "worker", workerID, "tweet_id", item.Tweet.ID, "error", err)
		}
	}
}

func runHealthServer(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	slog.Info("health server listening", "port", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		slog.Error("health server failed", "error", err)
	}
}
