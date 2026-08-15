package feed

import (
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

// RSSSource polls a fixed list of RSS feed URLs using Colly and parses RSS 2.0
// <item> entries. Used when FEATURE_LIVE_FEED=true. Financial news publishers
// (Reuters, MarketWatch, Business Wire, Benzinga, Yahoo Finance, ...) all
// publish standard RSS 2.0 feeds, which sidesteps X/Twitter's scraping ToS
// entirely.
type RSSSource struct {
	urls      []string
	collector *colly.Collector
}

func NewRSSSource(urls []string) *RSSSource {
	c := colly.NewCollector(
		colly.UserAgent("stock-alert-app/1.0 (+https://example.com/bot)"),
	)
	c.AllowURLRevisit = true // we poll the same feed URLs on every interval
	c.SetRequestTimeout(10 * time.Second)
	_ = c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 2,
		Delay:       500 * time.Millisecond,
	})

	return &RSSSource{urls: urls, collector: c}
}

func (s *RSSSource) Name() string { return "rss" }

type rssDocument struct {
	Channel struct {
		Title string    `xml:"title"`
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	GUID        string `xml:"guid"`
	PubDate     string `xml:"pubDate"`
}

var pubDateLayouts = []string{
	time.RFC1123Z,
	time.RFC1123,
	"2006-01-02T15:04:05Z07:00", // RFC3339, some feeds (Atom-ish) use this
}

func parsePubDate(v string) time.Time {
	for _, layout := range pubDateLayouts {
		if t, err := time.Parse(layout, strings.TrimSpace(v)); err == nil {
			return t
		}
	}
	return time.Now()
}

func (s *RSSSource) Poll(ctx context.Context) ([]Item, error) {
	var items []Item
	var firstErr error

	c := s.collector.Clone()
	c.OnResponse(func(r *colly.Response) {
		var doc rssDocument
		if err := xml.Unmarshal(r.Body, &doc); err != nil {
			slog.Warn("rss: failed to parse feed", "url", r.Request.URL.String(), "error", err)
			return
		}
		sourceName := doc.Channel.Title
		if sourceName == "" {
			sourceName = r.Request.URL.Hostname()
		}
		for _, it := range doc.Channel.Items {
			externalID := it.GUID
			if externalID == "" {
				externalID = it.Link
			}
			if externalID == "" {
				continue // nothing usable to dedup on
			}
			pub := parsePubDate(it.PubDate)
			items = append(items, Item{
				ExternalID:    externalID,
				SourceAccount: sourceName,
				Title:         strings.TrimSpace(it.Title),
				Content:       strings.TrimSpace(stripTags(it.Description)),
				URL:           it.Link,
				PublishedAt:   pub,
			})
		}
	})
	c.OnError(func(r *colly.Response, err error) {
		if firstErr == nil {
			firstErr = fmt.Errorf("fetching %s: %w", r.Request.URL, err)
		}
		slog.Warn("rss: request failed", "url", r.Request.URL.String(), "error", err)
	})

	for _, url := range s.urls {
		if ctx.Err() != nil {
			return items, ctx.Err()
		}
		if err := c.Visit(url); err != nil {
			slog.Warn("rss: visit failed", "url", url, "error", err)
		}
	}
	c.Wait()

	return items, nil
}

// stripTags does a minimal strip of HTML tags some feeds embed in <description>.
func stripTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
