// Package feed produces new financial news items to analyze. Two Source
// implementations are provided: MockSource (local fixture, used until a real
// feed is configured) and RSSSource (Colly-driven RSS polling).
package feed

import (
	"context"
	"time"
)

// Item is a single news item before it has been written to the tweets table.
type Item struct {
	ExternalID    string // unique id for dedup: RSS guid, falling back to link
	SourceAccount string // feed/publisher name
	Title         string
	Content       string
	URL           string
	PublishedAt   time.Time
}

// Source produces a batch of items on each Poll call. Implementations are
// responsible for their own internal state; callers rely on DB-level
// deduplication (unique constraint on tweets.external_id) as the source of
// truth, so a Source is free to return items it has returned before.
type Source interface {
	Name() string
	Poll(ctx context.Context) ([]Item, error)
}
